"""Config-flow scope changes stay narrow and preserve the existing entry."""

import sys
from types import SimpleNamespace
from unittest.mock import AsyncMock, MagicMock

import pytest

sys.path.insert(0, "homeassistant")

from custom_components.icloud_reminders_bridge.config_flow import (  # noqa: E402
    ICloudRemindersBridgeConfigFlow,
)
from custom_components.icloud_reminders_bridge.const import (  # noqa: E402
    CONF_ALLOWED_CALENDAR_IDS,
    CONF_ALLOWED_LIST_IDS,
    CONF_PAIRING_TOKEN,
)


@pytest.mark.asyncio
async def test_reconfigure_updates_scopes_without_replacing_entry():
    flow = ICloudRemindersBridgeConfigFlow()
    entry = SimpleNamespace(
        data={CONF_ALLOWED_LIST_IDS: ["old-list"], CONF_ALLOWED_CALENDAR_IDS: ["old-calendar"]}
    )
    result = {"type": "abort", "reason": "reconfigure_successful"}
    flow._get_reconfigure_entry = MagicMock(return_value=entry)
    flow.async_update_reload_and_abort = MagicMock(return_value=result)

    assert await flow.async_step_reconfigure(
        {
            CONF_ALLOWED_LIST_IDS: "new-list\nnew-list\n second-list ",
            CONF_ALLOWED_CALENDAR_IDS: "new-calendar",
        }
    ) == result
    flow.async_update_reload_and_abort.assert_called_once_with(
        entry,
        data_updates={
            CONF_ALLOWED_LIST_IDS: ["new-list", "second-list"],
            CONF_ALLOWED_CALENDAR_IDS: ["new-calendar"],
        },
    )


@pytest.mark.asyncio
async def test_reconfigure_rejects_empty_scopes():
    flow = ICloudRemindersBridgeConfigFlow()
    entry = SimpleNamespace(data={CONF_ALLOWED_LIST_IDS: [], CONF_ALLOWED_CALENDAR_IDS: []})
    flow._get_reconfigure_entry = MagicMock(return_value=entry)
    flow.async_show_form = MagicMock(return_value={"type": "form"})

    assert await flow.async_step_reconfigure(
        {CONF_ALLOWED_LIST_IDS: "\n", CONF_ALLOWED_CALENDAR_IDS: "  "}
    ) == {"type": "form"}
    assert flow.async_show_form.call_args.kwargs["errors"] == {"base": "no_lists"}


@pytest.mark.asyncio
async def test_reconfigure_rotates_token_without_replacing_entry():
    flow = ICloudRemindersBridgeConfigFlow()
    entry = SimpleNamespace(data={CONF_ALLOWED_LIST_IDS: ["old-list"], CONF_ALLOWED_CALENDAR_IDS: []})
    result = {"type": "abort", "reason": "reconfigure_successful"}
    flow._get_reconfigure_entry = MagicMock(return_value=entry)
    flow.async_update_reload_and_abort = MagicMock(return_value=result)
    token = "a" * 64

    assert await flow.async_step_reconfigure(
        {
            CONF_PAIRING_TOKEN: token,
            CONF_ALLOWED_LIST_IDS: "new-list",
            CONF_ALLOWED_CALENDAR_IDS: "",
        }
    ) == result
    call = flow.async_update_reload_and_abort.call_args
    assert call.args == (entry,)
    assert call.kwargs["data_updates"][CONF_PAIRING_TOKEN] == token
    assert call.kwargs["unique_id"]


@pytest.mark.asyncio
async def test_initial_setup_rejects_blank_identity():
    flow = ICloudRemindersBridgeConfigFlow()
    flow.async_show_form = MagicMock(return_value={"type": "form"})

    assert await flow.async_step_user(
        {
            "name": " ",
            "bridge_id": " ",
            CONF_PAIRING_TOKEN: "a" * 64,
            CONF_ALLOWED_LIST_IDS: "list",
            CONF_ALLOWED_CALENDAR_IDS: "",
        }
    ) == {"type": "form"}
    assert flow.async_show_form.call_args.kwargs["errors"] == {"name": "invalid_name"}
