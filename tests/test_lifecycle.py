"""Config-entry lifecycle tests for webhook registration and token rotation."""

from types import SimpleNamespace
from unittest.mock import AsyncMock, MagicMock, patch

import pytest

from custom_components.icloud_reminders_bridge import async_setup_entry, async_unload_entry


class FakeRuntime:
    webhook_token = "old-token"

    def __init__(self, *_args):
        self.async_load = AsyncMock()


@pytest.mark.asyncio
async def test_unload_unregisters_token_used_by_old_runtime():
    entry = SimpleNamespace(
        entry_id="entry",
        title="Fruit Forwarder",
        data={"pairing_token": "new-token"},
        runtime_data=FakeRuntime(),
    )
    hass = SimpleNamespace(
        config_entries=SimpleNamespace(
            async_unload_platforms=AsyncMock(return_value=True),
        )
    )

    with patch("custom_components.icloud_reminders_bridge.webhook.async_unregister") as unregister:
        assert await async_unload_entry(hass, entry)
    unregister.assert_called_once_with(hass, "old-token")


@pytest.mark.asyncio
async def test_setup_registers_runtime_token():
    entry = SimpleNamespace(
        entry_id="entry",
        title="Fruit Forwarder",
        data={"pairing_token": "new-token"},
        runtime_data=None,
        async_on_unload=MagicMock(),
    )
    hass = SimpleNamespace(
        config_entries=SimpleNamespace(async_forward_entry_setups=AsyncMock()),
    )

    with (
        patch("custom_components.icloud_reminders_bridge.BridgeRuntime", FakeRuntime),
        patch("custom_components.icloud_reminders_bridge.webhook.async_register") as register,
    ):
        assert await async_setup_entry(hass, entry)
    assert entry.runtime_data.webhook_token == "old-token"
    assert register.call_args.args[3] == "old-token"


@pytest.mark.asyncio
async def test_setup_failure_unregisters_webhook():
    entry = SimpleNamespace(
        entry_id="entry",
        title="Fruit Forwarder",
        data={"pairing_token": "new-token"},
        runtime_data=None,
        async_on_unload=MagicMock(),
    )
    hass = SimpleNamespace(
        config_entries=SimpleNamespace(
            async_forward_entry_setups=AsyncMock(side_effect=RuntimeError("platform failed")),
        )
    )

    with (
        patch("custom_components.icloud_reminders_bridge.BridgeRuntime", FakeRuntime),
        patch("custom_components.icloud_reminders_bridge.webhook.async_register"),
        patch("custom_components.icloud_reminders_bridge.webhook.async_unregister") as unregister,
    ):
        with pytest.raises(RuntimeError, match="platform failed"):
            await async_setup_entry(hass, entry)
    unregister.assert_called_once_with(hass, "old-token")
