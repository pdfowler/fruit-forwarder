"""Redacted Home Assistant diagnostics tests."""

import json
from types import SimpleNamespace

import pytest

from custom_components.icloud_reminders_bridge.diagnostics import (
    async_get_config_entry_diagnostics,
)
from test_runtime import runtime, snapshot


@pytest.mark.asyncio
async def test_diagnostics_exclude_household_data_and_credentials(tmp_path):
    bridge = runtime(tmp_path)
    await bridge.async_process_snapshot(snapshot())
    bridge.commands = [{"id": "secret-command-id", "list_id": "secret-list-id"}]
    bridge.entry.data.update(
        {
            "pairing_token": "secret-token",
            "allowed_list_ids": ["secret-list-id"],
            "allowed_calendar_ids": ["secret-calendar-id"],
        }
    )
    entry = SimpleNamespace(
        title="Household secret title",
        data=bridge.entry.data,
        runtime_data=bridge,
    )

    result = await async_get_config_entry_diagnostics(None, entry)
    encoded = json.dumps(result)

    assert result["runtime"]["list_count"] == 1
    assert result["runtime"]["item_count"] == 1
    assert result["runtime"]["pending_command_count"] == 1
    assert result["runtime"]["sync_status"] == "confirmed"
    assert result["runtime"]["calendar_sync_status"] == "unknown"
    for secret in (
        "secret-token",
        "secret-list-id",
        "secret-calendar-id",
        "secret-command-id",
        "Household secret title",
    ):
        assert secret not in encoded
