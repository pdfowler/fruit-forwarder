from copy import deepcopy
from datetime import UTC, datetime, timedelta

import pytest
from homeassistant.exceptions import HomeAssistantError

from custom_components.icloud_reminders_bridge.calendar import BridgeCalendar
from custom_components.icloud_reminders_bridge.runtime import ProtocolError
from test_runtime import runtime, snapshot


def calendar_payload():
    payload = snapshot()
    payload["calendars"] = [{"id": "events", "name": "Events",
        "window_start": "2026-01-01T00:00:00Z", "window_end": "2026-04-01T00:00:00Z",
        "events": [{"uid": "one", "summary": "All day", "all_day": True,
                    "start": "2026-02-01", "end": "2026-02-02"}]}]
    return payload


@pytest.mark.asyncio
async def test_calendar_snapshot_entity_and_restart(tmp_path):
    bridge = runtime(tmp_path)
    bridge.entry.data["allowed_calendar_ids"] = ["events"]
    await bridge.async_process_snapshot(calendar_payload())
    entity = BridgeCalendar(bridge, "events")
    events = await entity.async_get_events(bridge.hass,
        datetime(2026, 1, 31, tzinfo=UTC), datetime(2026, 2, 3, tzinfo=UTC))
    assert len(events) == 1
    assert events[0].all_day
    assert events[0].end.isoformat() == "2026-02-02"
    assert entity.supported_features == 0
    assert entity.should_poll
    with pytest.raises(HomeAssistantError):
        await entity.async_get_events(bridge.hass,
            datetime(2025, 1, 1, tzinfo=UTC), datetime(2026, 2, 3, tzinfo=UTC))
    stored = deepcopy(bridge._store.async_save.call_args.args[0])
    restored = runtime(tmp_path)
    restored.entry.data["allowed_calendar_ids"] = ["events"]
    restored._store.async_load.return_value = stored
    await restored.async_load()
    assert restored.calendars == bridge.calendars
    revoked = runtime(tmp_path)
    revoked._store.async_load.return_value = stored
    await revoked.async_load()
    assert revoked.calendars == {}


@pytest.mark.asyncio
@pytest.mark.parametrize("problem", ["scope", "date", "duplicate", "boolean"])
async def test_invalid_calendar_snapshot_is_atomic(tmp_path, problem):
    bridge = runtime(tmp_path)
    bridge.entry.data["allowed_calendar_ids"] = ["events"]
    await bridge.async_process_snapshot(calendar_payload())
    previous = deepcopy(bridge.calendars)
    payload = calendar_payload()
    calendar = payload["calendars"][0]
    if problem == "scope": calendar["id"] = "outside"
    if problem == "date": calendar["events"][0]["end"] = "bad"
    if problem == "duplicate": calendar["events"].append(deepcopy(calendar["events"][0]))
    if problem == "boolean": calendar["events"][0]["all_day"] = "false"
    with pytest.raises(ProtocolError):
        await bridge.async_process_snapshot(payload)
    assert bridge.calendars == previous


@pytest.mark.asyncio
async def test_calendar_save_failure_rolls_back(tmp_path):
    bridge = runtime(tmp_path)
    bridge.entry.data["allowed_calendar_ids"] = ["events"]
    bridge._store.async_save.side_effect = OSError("disk full")
    with pytest.raises(OSError):
        await bridge.async_process_snapshot(calendar_payload())
    assert bridge.calendars == {}


@pytest.mark.asyncio
async def test_renamed_calendar_updates_friendly_name_without_changing_identity(tmp_path):
    bridge = runtime(tmp_path)
    bridge.entry.data["allowed_calendar_ids"] = ["events"]
    await bridge.async_process_snapshot(calendar_payload())
    entity = BridgeCalendar(bridge, "events")
    unique_id = entity.unique_id

    renamed = calendar_payload()
    renamed["calendars"][0]["name"] = "Renamed Events"
    await bridge.async_process_snapshot(renamed)
    entity._refresh_name()

    assert entity.name == "Renamed Events"
    assert entity.unique_id == unique_id


@pytest.mark.asyncio
async def test_calendar_becomes_unavailable_when_calendar_snapshot_is_stale(tmp_path):
    bridge = runtime(tmp_path)
    bridge.entry.data["allowed_calendar_ids"] = ["events"]
    await bridge.async_process_snapshot(calendar_payload())
    entity = BridgeCalendar(bridge, "events")
    bridge.calendar_last_sync = (datetime.now(UTC) - timedelta(minutes=11)).isoformat()

    assert not entity.available
    assert entity.extra_state_attributes["sync_status"] == "stale"


def test_calendar_validation_bounds_identifiers_and_text():
    from custom_components.icloud_reminders_bridge.calendar_data import validate_calendars

    base = calendar_payload()["calendars"][0]
    for field, value in (("id", "x" * 4097), ("name", "x" * 4097)):
        invalid = deepcopy(base)
        invalid[field] = value
        with pytest.raises(ValueError):
            validate_calendars([invalid], {"events"})

    invalid_event = deepcopy(base)
    invalid_event["events"][0]["uid"] = "x" * 4097
    with pytest.raises(ValueError):
        validate_calendars([invalid_event], {"events"})
