"""HTTP boundary tests for the Home Assistant webhook callback."""

import json
from types import SimpleNamespace
from unittest.mock import AsyncMock, MagicMock, patch

import pytest
import pytest_asyncio

from custom_components.icloud_reminders_bridge import async_setup_entry
from custom_components.icloud_reminders_bridge.const import MAX_PAYLOAD_BYTES


class FakeRuntime:
    webhook_token = "pairing-token"

    def __init__(self, *_args):
        self.async_load = AsyncMock()
        self.async_process_snapshot = AsyncMock(return_value={"version": 1, "commands": []})


class FakeContent:
    def __init__(self, chunks):
        self.chunks = chunks

    async def iter_chunked(self, _size):
        for chunk in self.chunks:
            yield chunk


def request(chunks, content_length=None):
    return SimpleNamespace(content_length=content_length, content=FakeContent(chunks))


@pytest_asyncio.fixture
async def handler():
    entry = SimpleNamespace(
        entry_id="entry",
        title="Fruit Forwarder",
        data={"pairing_token": "pairing-token"},
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
        await async_setup_entry(hass, entry)
    yield register.call_args.args[4], entry.runtime_data


@pytest.mark.asyncio
async def test_webhook_accepts_bounded_json(handler):
    callback, runtime = handler
    body = json.dumps({"ok": True}).encode()
    response = await callback(None, "pairing-token", request([body], len(body)))
    assert response.status == 200
    assert json.loads(response.text) == {"version": 1, "commands": []}
    runtime.async_process_snapshot.assert_awaited_once_with({"ok": True})


@pytest.mark.asyncio
async def test_webhook_rejects_oversized_and_malformed_payloads(handler):
    callback, runtime = handler
    response = await callback(None, "pairing-token", request([b"{}"], MAX_PAYLOAD_BYTES + 1))
    assert response.status == 413
    runtime.async_process_snapshot.assert_not_awaited()

    response = await callback(None, "pairing-token", request([b"not-json"]))
    assert response.status == 400

    response = await callback(
        None,
        "pairing-token",
        request([b"x" * (MAX_PAYLOAD_BYTES // 2), b"y" * (MAX_PAYLOAD_BYTES // 2 + 1)]),
    )
    assert response.status == 413
