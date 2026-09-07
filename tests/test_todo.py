"""Todo entity capability and mutation-boundary tests."""

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
    await entity.async_create_todo_item(
        TodoItem(summary="Queued", status=TodoItemStatus.NEEDS_ACTION)
    )
    assert bridge.commands[-1]["action"] == "create"
