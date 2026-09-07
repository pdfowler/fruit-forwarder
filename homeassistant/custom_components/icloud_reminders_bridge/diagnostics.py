"""Redacted Home Assistant diagnostics for Fruit Forwarder."""

from __future__ import annotations

from typing import Any

from homeassistant.config_entries import ConfigEntry
from homeassistant.core import HomeAssistant

from .runtime import BridgeRuntime


async def async_get_config_entry_diagnostics(
    _hass: HomeAssistant, entry: ConfigEntry[BridgeRuntime]
) -> dict[str, Any]:
    """Return operational counts without household data or credentials."""
    runtime = entry.runtime_data
    lists = list(runtime.lists.values())
    return {
        "integration": {
            "name": "Fruit Forwarder",
            "entry_title_configured": bool(entry.title),
            "bridge_id_configured": bool(entry.data.get("bridge_id")),
            "pairing_token_configured": bool(entry.data.get("pairing_token")),
            "allowed_list_count": len(entry.data.get("allowed_list_ids", [])),
            "allowed_calendar_count": len(entry.data.get("allowed_calendar_ids", [])),
        },
        "runtime": {
            "list_count": len(lists),
            "read_only_list_count": sum(bool(item.get("read_only")) for item in lists),
            "item_count": sum(len(item.get("items", [])) for item in lists),
            "calendar_count": len(runtime.calendars),
            "pending_command_count": len(runtime.commands),
            "last_sync": runtime.last_sync,
            "calendar_last_sync": runtime.calendar_last_sync,
        },
    }
