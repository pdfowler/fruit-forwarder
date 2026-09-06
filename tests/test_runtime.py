"""Exercise the actual HA runtime with isolated storage, never a live HA instance."""

import asyncio
from copy import deepcopy
from types import SimpleNamespace
from unittest.mock import AsyncMock

import pytest
from homeassistant.core import HomeAssistant

from custom_components.icloud_reminders_bridge.runtime import BridgeRuntime, ProtocolError


def snapshot(status="needs_action", applied=None):
    return {
        "version": 1, "bridge_id": "test",
        "lists": [{"id": "allowed", "name": "Tasks", "items": [
            {"uid": "item", "summary": "Task", "status": status}
        ]}],
        "applied_command_ids": applied or [],
    }


def runtime(tmp_path):
    hass = HomeAssistant(str(tmp_path))
    entry = SimpleNamespace(entry_id="test", data={
        "bridge_id": "test", "allowed_list_ids": ["allowed"]
    })
    result = BridgeRuntime(hass, entry)
    result._store = SimpleNamespace(async_save=AsyncMock(), async_load=AsyncMock(return_value=None))
    return result


@pytest.mark.asyncio
async def test_command_survives_restart_and_is_acknowledged(tmp_path):
    bridge = runtime(tmp_path)
    await bridge.async_process_snapshot(snapshot())
    command_id = await bridge.async_queue_command("complete", "allowed", {"uid": "item"})
    stored = deepcopy(bridge._store.async_save.call_args.args[0])
    restored = runtime(tmp_path)
    restored._store.async_load.return_value = stored
    await restored.async_load()
    response = await restored.async_process_snapshot(snapshot())
    assert response["commands"][0]["id"] == command_id
    assert restored.lists["allowed"]["items"][0]["status"] == "completed"
    response = await restored.async_process_snapshot(snapshot("completed", [command_id]))
    assert response["commands"] == []
    assert restored.commands == []


@pytest.mark.asyncio
@pytest.mark.parametrize("failure", [OSError("disk full"), asyncio.CancelledError()])
async def test_failed_command_save_rolls_back_and_does_not_notify(tmp_path, failure):
    bridge = runtime(tmp_path)
    await bridge.async_process_snapshot(snapshot())
    notifications = []
    bridge.subscribe(lambda: notifications.append(True))
    bridge._store.async_save.side_effect = failure
    with pytest.raises(type(failure)):
        await bridge.async_queue_command("complete", "allowed", {"uid": "item"})
    assert bridge.commands == []
    assert bridge.lists["allowed"]["items"][0]["status"] == "needs_action"
    assert notifications == []


@pytest.mark.asyncio
async def test_failed_ack_save_preserves_pending_command(tmp_path):
    bridge = runtime(tmp_path)
    await bridge.async_process_snapshot(snapshot())
    command_id = await bridge.async_queue_command("complete", "allowed", {"uid": "item"})
    previous = deepcopy((bridge.lists, bridge.commands, bridge.last_sync))
    bridge._store.async_save.side_effect = OSError("disk full")
    with pytest.raises(OSError):
        await bridge.async_process_snapshot(snapshot("completed", [command_id]))
    assert (bridge.lists, bridge.commands, bridge.last_sync) == previous


@pytest.mark.asyncio
async def test_out_of_scope_snapshot_does_not_change_state(tmp_path):
    bridge = runtime(tmp_path)
    await bridge.async_process_snapshot(snapshot())
    previous = deepcopy(bridge.lists)
    invalid = snapshot()
    invalid["lists"][0]["id"] = "outside"
    with pytest.raises(ProtocolError):
        await bridge.async_process_snapshot(invalid)
    assert bridge.lists == previous


@pytest.mark.asyncio
async def test_revoked_lists_and_commands_do_not_return_after_restart(tmp_path):
    bridge = runtime(tmp_path)
    await bridge.async_process_snapshot(snapshot())
    await bridge.async_queue_command("complete", "allowed", {"uid": "item"})
    stored = deepcopy(bridge._store.async_save.call_args.args[0])
    restored = runtime(tmp_path)
    restored.entry.data["allowed_list_ids"] = []
    restored._store.async_load.return_value = stored
    await restored.async_load()
    assert restored.lists == {}
    assert restored.commands == []
    cleaned = deepcopy(restored._store.async_save.call_args.args[0])
    readded = runtime(tmp_path)
    readded._store.async_load.return_value = cleaned
    await readded.async_load()
    assert readded.lists == {}
    assert readded.commands == []


@pytest.mark.asyncio
@pytest.mark.parametrize("condition", ["missing", "read_only"])
async def test_unwritable_lists_pause_dispatch_without_losing_edits(tmp_path, condition):
    bridge = runtime(tmp_path)
    await bridge.async_process_snapshot(snapshot())
    command_id = await bridge.async_queue_command("complete", "allowed", {"uid": "item"})
    unavailable = snapshot()
    if condition == "missing":
        unavailable["lists"] = []
    else:
        unavailable["lists"][0]["read_only"] = True
    response = await bridge.async_process_snapshot(unavailable)
    assert response["commands"] == []
    assert bridge.commands[0]["id"] == command_id
    response = await bridge.async_process_snapshot(snapshot())
    assert response["commands"][0]["id"] == command_id


@pytest.mark.asyncio
async def test_revoked_pending_edit_is_not_returned_by_next_snapshot(tmp_path):
    bridge = runtime(tmp_path)
    await bridge.async_process_snapshot(snapshot())
    await bridge.async_queue_command("complete", "allowed", {"uid": "item"})
    bridge.entry.data["allowed_list_ids"] = []
    empty = snapshot()
    empty["lists"] = []
    response = await bridge.async_process_snapshot(empty)
    assert response["commands"] == []
    assert bridge.commands == []
    assert bridge.lists == {}
