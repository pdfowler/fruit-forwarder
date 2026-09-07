"""Runtime state and the narrow bridge protocol."""

from __future__ import annotations

import asyncio
from collections.abc import Callable
from copy import deepcopy
from contextlib import asynccontextmanager
from datetime import UTC, datetime
import json
import uuid
from typing import Any

from homeassistant.config_entries import ConfigEntry
from homeassistant.core import HomeAssistant, callback
from homeassistant.helpers.storage import Store

from .const import (
    CAPABILITY_CALENDARS,
    CAPABILITY_COMMAND_QUEUE,
    CAPABILITY_QUEUE_EPOCH,
    CAPABILITY_REMINDERS,
    CONF_ALLOWED_CALENDAR_IDS,
    CONF_ALLOWED_LIST_IDS,
    CONF_PAIRING_TOKEN,
    CONF_BRIDGE_ID,
    MAX_PAYLOAD_BYTES,
    MAX_ITEMS,
    MAX_LISTS,
    MAX_COMMANDS,
    MAX_STRING_LENGTH,
    PROTOCOL_VERSION,
)
from .calendar_data import validate_calendars

UpdateCallback = Callable[[], None]


class ProtocolError(ValueError):
    """Raised for an invalid or out-of-scope bridge payload."""


class BridgeRuntime:
    """Persist snapshots and an idempotent command queue for one bridge."""

    def __init__(self, hass: HomeAssistant, entry: ConfigEntry) -> None:
        self.hass = hass
        self.entry = entry
        # Keep the token used for webhook registration stable for the lifetime
        # of this runtime. During a reconfigure/reload HA may update
        # entry.data before unloading the old runtime; using entry.data during
        # unload would leave the old webhook registered after token rotation.
        self.webhook_token = entry.data.get(CONF_PAIRING_TOKEN, "")
        self._store: Store[dict[str, Any]] = Store(
            hass, 1, f"icloud_reminders_bridge.{entry.entry_id}"
        )
        self._lock = asyncio.Lock()
        self._listeners: list[UpdateCallback] = []
        self.lists: dict[str, dict[str, Any]] = {}
        self.calendars: dict[str, dict[str, Any]] = {}
        self.commands: list[dict[str, Any]] = []
        self.queue_epoch: str | None = None
        self.last_sync: str | None = None
        self.calendar_last_sync: str | None = None

    async def async_load(self) -> None:
        """Restore the last confirmed snapshot and pending commands."""
        stored = await self._store.async_load() or {}
        stored_epoch = stored.get("queue_epoch")
        if stored_epoch is not None and (
            not isinstance(stored_epoch, str)
            or not stored_epoch.strip()
            or len(stored_epoch) > 128
        ):
            raise ProtocolError("Stored queue epoch is invalid")
        self.queue_epoch = stored_epoch or uuid.uuid4().hex
        allowed_ids = set(self.entry.data[CONF_ALLOWED_LIST_IDS])
        calendar_ids = set(self.entry.data.get(CONF_ALLOWED_CALENDAR_IDS, []))
        self.calendars = validate_calendars([
            c for c in stored.get("calendars", [])
            if isinstance(c, dict) and c.get("id") in calendar_ids
        ], calendar_ids)
        self.lists = {
            item["id"]: item
            for item in stored.get("lists", [])
            if isinstance(item, dict) and isinstance(item.get("id"), str)
            and item["id"] in allowed_ids
        }
        raw_commands = stored.get("commands", [])
        if not isinstance(raw_commands, list) or len(raw_commands) > MAX_COMMANDS:
            raise ProtocolError("Stored command queue is invalid or exceeds its bound")
        self.commands = []
        for command in raw_commands:
            if not isinstance(command, dict):
                raise ProtocolError("Stored command must be an object")
            command_id = command.get("id")
            list_id = command.get("list_id")
            action = command.get("action")
            item = command.get("item")
            if (
                not isinstance(command_id, str)
                or not command_id.strip()
                or len(command_id) > 256
                or not isinstance(list_id, str)
                or not list_id.strip()
                or len(list_id) > MAX_STRING_LENGTH
                or action not in {"create", "update", "complete", "reopen"}
            ):
                raise ProtocolError("Stored command has invalid identity or action")
            _validate_command_item(action, item)
            if list_id in allowed_ids:
                self.commands.append(command)
        self.last_sync = stored.get("last_sync")
        self.calendar_last_sync = stored.get("calendar_last_sync")
        if (stored_epoch != self.queue_epoch
                or len(self.lists) != len(stored.get("lists", []))
                or len(self.commands) != len(stored.get("commands", []))
                or len(self.calendars) != len(stored.get("calendars", []))):
            # Persist revocation so re-adding a list cannot resurrect stale edits.
            await self._async_save()

    @callback
    def subscribe(self, listener: UpdateCallback) -> Callable[[], None]:
        """Subscribe to snapshots and optimistic command updates."""
        self._listeners.append(listener)

        @callback
        def remove_listener() -> None:
            if listener in self._listeners:
                self._listeners.remove(listener)

        return remove_listener

    async def async_process_snapshot(
        self, payload: dict[str, Any]
    ) -> dict[str, Any]:
        """Validate and store a bridge snapshot, then return queued edits."""
        lists, applied_ids = self._validate_snapshot(payload)
        try:
            calendars = None
            if "calendars" in payload:
                calendars = validate_calendars(payload.get("calendars", []),
                    set(self.entry.data.get(CONF_ALLOWED_CALENDAR_IDS, [])))
        except ValueError as err:
            raise ProtocolError(str(err)) from err
        async with self._transaction():
            if self.queue_epoch is None:
                self.queue_epoch = uuid.uuid4().hex
            if calendars is not None:
                self.calendars = calendars
                self.calendar_last_sync = datetime.now(UTC).isoformat()
            allowed_ids = set(self.entry.data[CONF_ALLOWED_LIST_IDS])
            self.commands = [
                command for command in self.commands
                if command.get("id") not in applied_ids
                and command.get("list_id") in allowed_ids
            ]
            self.lists = {item["id"]: item for item in lists}
            for command in self.commands:
                if command["list_id"] in self.lists:
                    self._apply_optimistic(command)
            self.last_sync = datetime.now(UTC).isoformat()
            response = {
                "version": PROTOCOL_VERSION,
                "capabilities": [
                    CAPABILITY_REMINDERS,
                    CAPABILITY_COMMAND_QUEUE,
                    CAPABILITY_QUEUE_EPOCH,
                    *(
                        [CAPABILITY_CALENDARS]
                        if CONF_ALLOWED_CALENDAR_IDS in self.entry.data
                        else []
                    ),
                ],
                "queue_epoch": self.queue_epoch,
                # Keep temporarily unavailable/read-only edits queued, but do
                # not ask EventKit to execute them until the list is writable.
                "commands": deepcopy([
                    command for command in self.commands
                    if command["list_id"] in self.lists
                    and not self.lists[command["list_id"]]["read_only"]
                ]),
            }
            if len(json.dumps(response).encode("utf-8")) > MAX_PAYLOAD_BYTES:
                raise ProtocolError("Queued command response exceeds the payload bound")
        self._notify()
        return response

    async def async_queue_command(
        self, action: str, list_id: str, item: dict[str, Any]
    ) -> str:
        """Persist a scoped command before returning success to HA."""
        if action not in {"create", "update", "complete", "reopen"}:
            raise ProtocolError(f"Unsupported action: {action}")
        _validate_command_item(action, item)
        if list_id not in self.lists:
            raise ProtocolError("Reminder list is not available from the bridge")
        command_id = uuid.uuid4().hex
        command = {
            "id": command_id,
            "action": action,
            "list_id": list_id,
            "item": deepcopy(item),
        }
        async with self._transaction():
            reminder_list = self.lists.get(list_id)
            if reminder_list is None or list_id not in self.entry.data[CONF_ALLOWED_LIST_IDS]:
                raise ProtocolError("Reminder list is outside the configured allowlist")
            if reminder_list.get("read_only"):
                raise ProtocolError("Reminder list is read-only")
            if len(self.commands) >= MAX_COMMANDS:
                raise ProtocolError("Reminder command queue is full; wait for a sync")
            if action != "create":
                uid = item.get("uid", "")
                if uid.startswith("pending:"):
                    raise ProtocolError("This new reminder has not synced yet; retry after sync")
                if not any(existing["uid"] == uid for existing in reminder_list["items"]):
                    raise ProtocolError("Reminder no longer exists in this list")
            self.commands.append(command)
            self._apply_optimistic(command)
        self._notify()
        return command_id

    def _validate_snapshot(
        self, payload: dict[str, Any]
    ) -> tuple[list[dict[str, Any]], set[str]]:
        if not isinstance(payload, dict):
            raise ProtocolError("JSON body must be an object")
        if payload.get("version") != PROTOCOL_VERSION:
            raise ProtocolError("Unsupported bridge protocol version")
        if payload.get("bridge_id") != self.entry.data[CONF_BRIDGE_ID]:
            raise ProtocolError("Bridge identifier does not match this entry")
        _validate_capabilities(payload.get("capabilities"), payload)
        raw_lists = payload.get("lists")
        if not isinstance(raw_lists, list) or len(raw_lists) > MAX_LISTS:
            raise ProtocolError("Invalid reminder list collection")
        allowed_ids = set(self.entry.data[CONF_ALLOWED_LIST_IDS])
        validated: list[dict[str, Any]] = []
        total_items = 0
        seen_ids: set[str] = set()
        for raw_list in raw_lists:
            if not isinstance(raw_list, dict):
                raise ProtocolError("Each reminder list must be an object")
            list_id = _required_string(raw_list, "id")
            name = _required_string(raw_list, "name")
            if list_id not in allowed_ids:
                raise ProtocolError(f"Reminder list {list_id!r} is outside the allowlist")
            if list_id in seen_ids:
                raise ProtocolError("Duplicate reminder list identifier")
            seen_ids.add(list_id)
            raw_items = raw_list.get("items")
            if not isinstance(raw_items, list):
                raise ProtocolError("Reminder list items must be an array")
            total_items += len(raw_items)
            if total_items > MAX_ITEMS:
                raise ProtocolError("Snapshot contains too many reminders")
            items = [_validate_item(item) for item in raw_items]
            validated.append(
                {
                    "id": list_id,
                    "name": name,
                    "source": str(raw_list.get("source") or ""),
                    "read_only": bool(raw_list.get("read_only", False)),
                    "items": items,
                }
            )
        applied = payload.get("applied_command_ids", [])
        if not isinstance(applied, list) or not all(
            isinstance(item, str) for item in applied
        ) or len(applied) > MAX_COMMANDS:
            raise ProtocolError("applied_command_ids must be a bounded array of strings")
        if any(not item.strip() or len(item) > 256 for item in applied):
            raise ProtocolError("applied command identifiers must be non-empty and bounded")
        return validated, set(applied)

    def _apply_optimistic(self, command: dict[str, Any]) -> None:
        reminder_list = self.lists[command["list_id"]]
        items = reminder_list["items"]
        item = deepcopy(command["item"])
        if command["action"] == "create":
            item["uid"] = f"pending:{command['id']}"
            item.setdefault("status", "needs_action")
            items.append(item)
            return
        for index, existing in enumerate(items):
            if existing["uid"] != item.get("uid"):
                continue
            if command["action"] == "complete":
                existing["status"] = "completed"
            elif command["action"] == "reopen":
                existing["status"] = "needs_action"
                existing["completed"] = None
            else:
                items[index] = item
            return

    @asynccontextmanager
    async def _transaction(self):
        """Publish mutations only after storage succeeds; rollback failed saves."""
        async with self._lock:
            previous = deepcopy((self.lists, self.commands, self.queue_epoch, self.last_sync, self.calendars, self.calendar_last_sync))
            try:
                yield
                await self._async_save()
            except BaseException:
                self.lists, self.commands, self.queue_epoch, self.last_sync, self.calendars, self.calendar_last_sync = previous
                raise

    async def _async_save(self) -> None:
        await self._store.async_save(
            {
                "lists": list(self.lists.values()),
                "calendars": list(self.calendars.values()),
                "commands": self.commands,
                "queue_epoch": self.queue_epoch,
                "last_sync": self.last_sync,
                "calendar_last_sync": self.calendar_last_sync,
            }
        )

    @callback
    def _notify(self) -> None:
        for listener in tuple(self._listeners):
            listener()


def _required_string(value: dict[str, Any], key: str) -> str:
    result = value.get(key)
    if not isinstance(result, str) or not result.strip() or len(result) > MAX_STRING_LENGTH:
        raise ProtocolError(f"{key} must be a non-empty string")
    return result


def _validate_capabilities(raw: Any, payload: dict[str, Any]) -> None:
    """Validate optional capability declarations without blocking extensions."""
    if raw is None:
        return
    if not isinstance(raw, list) or len(raw) > 16:
        raise ProtocolError("capabilities must be a bounded array of strings")
    seen: set[str] = set()
    for capability in raw:
        if (
            not isinstance(capability, str)
            or not capability.strip()
            or len(capability) > 64
            or capability in seen
        ):
            raise ProtocolError("capabilities must contain unique bounded names")
        seen.add(capability)
    if "calendars" in payload and "calendars" not in seen:
        raise ProtocolError("calendar data requires the calendars capability")


def _validate_command_item(action: str, item: Any) -> None:
    """Bound queued mutation fields before they enter HA storage."""
    if not isinstance(item, dict):
        raise ProtocolError("Command item must be an object")
    uid = item.get("uid")
    if action != "create" and (
        not isinstance(uid, str) or not uid.strip() or len(uid) > MAX_STRING_LENGTH
    ):
        raise ProtocolError("Mutation command requires a bounded uid")
    summary = item.get("summary")
    if action in {"create", "update"} and (
        not isinstance(summary, str) or not summary.strip() or len(summary) > 256
    ):
        raise ProtocolError("Command summary is empty or exceeds 256 characters")
    if summary is not None and (
        not isinstance(summary, str) or len(summary) > 256
    ):
        raise ProtocolError("Command summary exceeds 256 characters")
    status = item.get("status")
    if status is not None and status not in {"needs_action", "completed"}:
        raise ProtocolError("Command status must be needs_action or completed")
    for field, limit in (("description", 4096), ("due", MAX_STRING_LENGTH), ("completed", MAX_STRING_LENGTH)):
        value = item.get(field)
        if value is not None and (not isinstance(value, str) or len(value) > limit):
            raise ProtocolError(f"Command {field} is invalid or exceeds its bound")
    if len(json.dumps(item, ensure_ascii=False).encode("utf-8")) > 16 * 1024:
        raise ProtocolError("Command item exceeds 16 KiB")


def _validate_item(value: Any) -> dict[str, Any]:
    if not isinstance(value, dict):
        raise ProtocolError("Each reminder must be an object")
    uid = _required_string(value, "uid")
    summary = _required_string(value, "summary")
    status = value.get("status")
    if status not in {"needs_action", "completed"}:
        raise ProtocolError("Reminder status must be needs_action or completed")
    result: dict[str, Any] = {"uid": uid, "summary": summary, "status": status}
    for field in ("description", "due", "completed", "modified"):
        field_value = value.get(field)
        if field_value is not None and not isinstance(field_value, str):
            raise ProtocolError(f"Reminder {field} must be a string or null")
        if field_value is not None:
            if len(field_value) > MAX_STRING_LENGTH:
                raise ProtocolError(f"Reminder {field} exceeds the supported size")
            result[field] = field_value
    return result
