"""Home Assistant todo entities backed by the macOS EventKit bridge."""

from __future__ import annotations

from datetime import date, datetime
from typing import Any

from homeassistant.components.todo import TodoItem, TodoListEntity
from homeassistant.components.todo.const import TodoItemStatus, TodoListEntityFeature
from homeassistant.core import callback
from homeassistant.helpers.entity_platform import AddConfigEntryEntitiesCallback

from . import ICloudRemindersConfigEntry
from .const import DOMAIN
from .runtime import BridgeRuntime


async def async_setup_entry(
    _hass: Any,
    entry: ICloudRemindersConfigEntry,
    async_add_entities: AddConfigEntryEntitiesCallback,
) -> None:
    """Create one entity per allowlisted list received from the macOS bridge."""
    runtime = entry.runtime_data
    entities: dict[str, ICloudReminderTodoEntity] = {}

    @callback
    def refresh_entities() -> None:
        new_entities: list[ICloudReminderTodoEntity] = []
        for list_id in runtime.lists:
            if list_id not in entities:
                entity = ICloudReminderTodoEntity(runtime, list_id)
                entities[list_id] = entity
                new_entities.append(entity)
        if new_entities:
            async_add_entities(new_entities)
        for entity in entities.values():
            entity.async_refresh_from_runtime()

    entry.async_on_unload(runtime.subscribe(refresh_entities))
    refresh_entities()


class ICloudReminderTodoEntity(TodoListEntity):
    """An editable, deletion-disabled view of one EventKit reminder list."""

    _attr_has_entity_name = True
    _attr_should_poll = False
    _attr_supported_features = (
        TodoListEntityFeature.CREATE_TODO_ITEM
        | TodoListEntityFeature.UPDATE_TODO_ITEM
        | TodoListEntityFeature.SET_DUE_DATE_ON_ITEM
        | TodoListEntityFeature.SET_DUE_DATETIME_ON_ITEM
        | TodoListEntityFeature.SET_DESCRIPTION_ON_ITEM
    )

    def __init__(self, runtime: BridgeRuntime, list_id: str) -> None:
        self._runtime = runtime
        self._list_id = list_id
        reminder_list = runtime.lists[list_id]
        self._attr_name = reminder_list["name"]
        self._attr_unique_id = f"{runtime.entry.entry_id}:{list_id}"
        self._attr_device_info = {
            "identifiers": {(DOMAIN, runtime.entry.entry_id)},
            "name": runtime.entry.title,
            "manufacturer": "Fruit Forwarder",
            "model": "EventKit bridge",
        }
        self.async_refresh_from_runtime(write_state=False)

    @property
    def extra_state_attributes(self) -> dict[str, Any]:
        """Expose operational health without exposing item contents."""
        return {
            "last_sync": self._runtime.last_sync,
            "pending_commands": sum(
                command.get("list_id") == self._list_id
                for command in self._runtime.commands
            ),
            "source": self._runtime.lists.get(self._list_id, {}).get("source"),
        }

    @callback
    def async_refresh_from_runtime(self, write_state: bool = True) -> None:
        """Adopt the latest EventKit-confirmed or optimistic state."""
        reminder_list = self._runtime.lists.get(self._list_id)
        self._attr_available = reminder_list is not None
        self._attr_todo_items = (
            [_to_ha_item(item) for item in reminder_list["items"]]
            if reminder_list
            else None
        )
        if write_state and self.hass is not None:
            self.async_write_ha_state()

    async def async_create_todo_item(self, item: TodoItem) -> None:
        """Queue creation for the macOS bridge and expose it optimistically."""
        await self._runtime.async_queue_command(
            "create", self._list_id, _from_ha_item(item)
        )

    async def async_update_todo_item(self, item: TodoItem) -> None:
        """Queue an update, completion, or reopen for the macOS bridge."""
        current = next(
            (
                existing
                for existing in (self.todo_items or [])
                if existing.uid == item.uid
            ),
            None,
        )
        action = "update"
        if current is not None and _same_content(current, item):
            if item.status == TodoItemStatus.COMPLETED:
                action = "complete"
            elif item.status == TodoItemStatus.NEEDS_ACTION:
                action = "reopen"
        await self._runtime.async_queue_command(
            action, self._list_id, _from_ha_item(item)
        )


def _to_ha_item(item: dict[str, Any]) -> TodoItem:
    return TodoItem(
        uid=item["uid"],
        summary=item["summary"],
        status=TodoItemStatus(item["status"]),
        description=item.get("description"),
        due=_parse_datetime(item.get("due")),
        completed=_parse_datetime(item.get("completed")),
    )


def _from_ha_item(item: TodoItem) -> dict[str, Any]:
    status = item.status or TodoItemStatus.NEEDS_ACTION
    if isinstance(status, TodoItemStatus):
        status = status.value
    return {
        "uid": item.uid or "",
        "summary": item.summary or "",
        "status": status,
        "description": item.description or "",
        "due": _serialize_date(item.due),
        "completed": _serialize_date(item.completed),
    }


def _parse_datetime(value: str | None) -> date | datetime | None:
    if not value:
        return None
    if len(value) == 10:
        return date.fromisoformat(value)
    return datetime.fromisoformat(value.replace("Z", "+00:00"))


def _serialize_date(value: date | datetime | None) -> str | None:
    return value.isoformat() if value is not None else None


def _same_content(left: TodoItem, right: TodoItem) -> bool:
    return (
        left.uid == right.uid
        and left.summary == right.summary
        and left.description == right.description
        and left.due == right.due
    )
