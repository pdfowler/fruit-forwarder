"""Runtime state and the narrow bridge protocol."""

from __future__ import annotations

import asyncio
from collections.abc import Callable
from copy import deepcopy
from datetime import UTC, datetime
import uuid
from typing import Any

from homeassistant.config_entries import ConfigEntry
from homeassistant.core import HomeAssistant, callback
from homeassistant.helpers.storage import Store

from .const import (
    CONF_ALLOWED_LIST_IDS,
    CONF_BRIDGE_ID,
    MAX_ITEMS,
    MAX_LISTS,
    PROTOCOL_VERSION,
)

UpdateCallback = Callable[[], None]


class ProtocolError(ValueError):
    """Raised for an invalid or out-of-scope bridge payload."""


class BridgeRuntime:
    """Persist snapshots and an idempotent command queue for one bridge."""

    def __init__(self, hass: HomeAssistant, entry: ConfigEntry) -> None:
        self.hass = hass
        self.entry = entry
        self._store: Store[dict[str, Any]] = Store(
            hass, 1, f"icloud_reminders_bridge.{entry.entry_id}"
        )
        self._lock = asyncio.Lock()
        self._listeners: list[UpdateCallback] = []
        self.lists: dict[str, dict[str, Any]] = {}
        self.commands: list[dict[str, Any]] = []
        self.last_sync: str | None = None

    async def async_load(self) -> None:
        """Restore the last confirmed snapshot and pending commands."""
        stored = await self._store.async_load() or {}
        self.lists = {
            item["id"]: item
            for item in stored.get("lists", [])
            if isinstance(item, dict) and isinstance(item.get("id"), str)
        }
        self.commands = [
            item for item in stored.get("commands", []) if isinstance(item, dict)
        ]
        self.last_sync = stored.get("last_sync")

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
        async with self._lock:
            if applied_ids:
                self.commands = [
                    command
                    for command in self.commands
                    if command.get("id") not in applied_ids
                ]
            self.lists = {item["id"]: item for item in lists}
            for command in self.commands:
                if command["list_id"] in self.lists:
                    self._apply_optimistic(command)
            self.last_sync = datetime.now(UTC).isoformat()
            await self._async_save()
            response = {
                "version": PROTOCOL_VERSION,
                "commands": deepcopy(self.commands),
            }
        self._notify()
        return response

    async def async_queue_command(
        self, action: str, list_id: str, item: dict[str, Any]
    ) -> str:
        """Persist a scoped command before returning success to HA."""
        if action not in {"create", "update", "complete", "reopen"}:
            raise ProtocolError(f"Unsupported action: {action}")
        if list_id not in self.lists:
            raise ProtocolError("Reminder list is not available from the bridge")
        command_id = uuid.uuid4().hex
        command = {
            "id": command_id,
            "action": action,
            "list_id": list_id,
            "item": deepcopy(item),
        }
        async with self._lock:
            reminder_list = self.lists.get(list_id)
            if reminder_list is None or list_id not in self.entry.data[CONF_ALLOWED_LIST_IDS]:
                raise ProtocolError("Reminder list is outside the configured allowlist")
            if reminder_list.get("read_only"):
                raise ProtocolError("Reminder list is read-only")
            if action != "create":
                uid = item.get("uid", "")
                if uid.startswith("pending:"):
                    raise ProtocolError("This new reminder has not synced yet; retry after sync")
                if not any(existing["uid"] == uid for existing in reminder_list["items"]):
                    raise ProtocolError("Reminder no longer exists in this list")
            self.commands.append(command)
            self._apply_optimistic(command)
            await self._async_save()
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
        ):
            raise ProtocolError("applied_command_ids must be an array of strings")
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

    async def _async_save(self) -> None:
        await self._store.async_save(
            {
                "lists": list(self.lists.values()),
                "commands": self.commands,
                "last_sync": self.last_sync,
            }
        )

    @callback
    def _notify(self) -> None:
        for listener in tuple(self._listeners):
            listener()


def _required_string(value: dict[str, Any], key: str) -> str:
    result = value.get(key)
    if not isinstance(result, str) or not result.strip():
        raise ProtocolError(f"{key} must be a non-empty string")
    return result


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
            result[field] = field_value
    return result
