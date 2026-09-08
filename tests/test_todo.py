"""Todo entity capability and mutation-boundary tests."""

from datetime import UTC, datetime, timedelta

import pytest

from homeassistant.components.todo import TodoItem
from homeassistant.components.todo.const import TodoItemStatus, TodoListEntityFeature
from homeassistant.exceptions import HomeAssistantError

from custom_components.icloud_reminders_bridge.todo import ICloudReminderTodoEntity
from test_runtime import runtime, snapshot


@pytest.mark.asyncio
async def test_read_only_list_hides_mutation_features_and_rejects_writes(tmp_path):
    bridge = runtime(tmp_path)
    payload = snapshot()
    payload["lists"][0]["read_only"] = True
    await bridge.async_process_snapshot(payload)
    entity = ICloudReminderTodoEntity(bridge, "allowed")

    assert entity.supported_features == TodoListEntityFeature(0)
    with pytest.raises(HomeAssistantError, match="read-only"):
        await entity.async_create_todo_item(
            TodoItem(summary="Do not write", status=TodoItemStatus.NEEDS_ACTION)
        )
    with pytest.raises(HomeAssistantError, match="read-only"):
        await entity.async_update_todo_item(
            TodoItem(uid="item", summary="Do not write", status=TodoItemStatus.NEEDS_ACTION)
        )


@pytest.mark.asyncio
async def test_writable_list_exposes_mutation_features(tmp_path):
    bridge = runtime(tmp_path)
    await bridge.async_process_snapshot(snapshot())
    entity = ICloudReminderTodoEntity(bridge, "allowed")

    assert entity.supported_features & TodoListEntityFeature.CREATE_TODO_ITEM
    assert entity.should_poll
    await entity.async_create_todo_item(
        TodoItem(summary="Queued", status=TodoItemStatus.NEEDS_ACTION)
    )
    assert bridge.commands[-1]["action"] == "create"


@pytest.mark.asyncio
async def test_renamed_list_updates_friendly_name_without_changing_identity(tmp_path):
    bridge = runtime(tmp_path)
    await bridge.async_process_snapshot(snapshot())
    entity = ICloudReminderTodoEntity(bridge, "allowed")
    unique_id = entity.unique_id

    renamed = snapshot()
    renamed["lists"][0]["name"] = "Renamed Tasks"
    await bridge.async_process_snapshot(renamed)
    entity.async_refresh_from_runtime(write_state=False)

    assert entity.name == "Renamed Tasks"
    assert entity.unique_id == unique_id


@pytest.mark.asyncio
async def test_list_becomes_unavailable_when_snapshot_is_stale(tmp_path):
    bridge = runtime(tmp_path)
    await bridge.async_process_snapshot(snapshot())
    entity = ICloudReminderTodoEntity(bridge, "allowed")
    bridge.last_sync = (datetime.now(UTC) - timedelta(minutes=11)).isoformat()
    entity.async_refresh_from_runtime(write_state=False)

    assert not entity.available
    assert entity.extra_state_attributes["sync_status"] == "stale"
