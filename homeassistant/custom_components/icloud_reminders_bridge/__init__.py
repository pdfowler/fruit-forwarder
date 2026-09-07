"""Fruit Forwarder Home Assistant integration."""

from __future__ import annotations

import json

from aiohttp import web

from homeassistant.components import webhook
from homeassistant.config_entries import ConfigEntry
from homeassistant.core import HomeAssistant

from .const import (
    CONF_PAIRING_TOKEN,
    DOMAIN,
    MAX_PAYLOAD_BYTES,
    PLATFORMS,
)
from .runtime import BridgeRuntime, ProtocolError

type ICloudRemindersConfigEntry = ConfigEntry[BridgeRuntime]


async def async_setup_entry(
    hass: HomeAssistant, entry: ICloudRemindersConfigEntry
) -> bool:
    """Set up a configured EventKit bridge."""
    runtime = BridgeRuntime(hass, entry)
    await runtime.async_load()
    entry.runtime_data = runtime

    async def handle_webhook(
        _hass: HomeAssistant, _webhook_id: str, request: web.Request
    ) -> web.Response:
        if request.content_length is not None and request.content_length > MAX_PAYLOAD_BYTES:
            return web.json_response({"error": "payload_too_large"}, status=413)
        try:
            raw_body = bytearray()
            async for chunk in request.content.iter_chunked(64 * 1024):
                raw_body.extend(chunk)
                if len(raw_body) > MAX_PAYLOAD_BYTES:
                    return web.json_response(
                        {"error": "payload_too_large"}, status=413
                    )
            payload = json.loads(raw_body)
            response = await runtime.async_process_snapshot(payload)
        except (json.JSONDecodeError, ProtocolError, TypeError, ValueError) as err:
            return web.json_response({"error": "invalid_payload", "message": str(err)}, status=400)
        return web.json_response(response, headers={"Cache-Control": "no-store"})

    webhook.async_register(
        hass,
        DOMAIN,
        entry.title,
        entry.data[CONF_PAIRING_TOKEN],
        handle_webhook,
        local_only=True,
        allowed_methods={"POST"},
    )
    await hass.config_entries.async_forward_entry_setups(entry, PLATFORMS)
    return True


async def async_unload_entry(
    hass: HomeAssistant, entry: ICloudRemindersConfigEntry
) -> bool:
    """Unload the bridge and its scoped webhook."""
    unloaded = await hass.config_entries.async_unload_platforms(entry, PLATFORMS)
    if unloaded:
        webhook.async_unregister(hass, entry.data[CONF_PAIRING_TOKEN])
    return unloaded
