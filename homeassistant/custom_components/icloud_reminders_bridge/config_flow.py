"""Config flow for iCloud Reminders Bridge."""

from __future__ import annotations

import hashlib
import re
from typing import Any

import voluptuous as vol

from homeassistant import config_entries
from homeassistant.const import CONF_NAME
from homeassistant.helpers import selector

from .const import CONF_ALLOWED_CALENDAR_IDS, CONF_ALLOWED_LIST_IDS, CONF_BRIDGE_ID, CONF_PAIRING_TOKEN, DOMAIN

TOKEN_PATTERN = re.compile(r"^[A-Za-z0-9_-]{43,128}$")


class ICloudRemindersBridgeConfigFlow(config_entries.ConfigFlow, domain=DOMAIN):
    """Configure a macOS EventKit reminders bridge."""

    VERSION = 1

    async def async_step_user(
        self, user_input: dict[str, Any] | None = None
    ) -> config_entries.ConfigFlowResult:
        """Handle the initial setup form."""
        errors: dict[str, str] = {}
        if user_input is not None:
            token = user_input[CONF_PAIRING_TOKEN].strip()
            allowed_list_ids = _parse_allowed_list_ids(
                user_input[CONF_ALLOWED_LIST_IDS]
            )
            if not TOKEN_PATTERN.fullmatch(token):
                errors[CONF_PAIRING_TOKEN] = "invalid_token"
            elif not allowed_list_ids and not _parse_allowed_list_ids(user_input.get(CONF_ALLOWED_CALENDAR_IDS, "")):
                errors[CONF_ALLOWED_LIST_IDS] = "no_lists"
            else:
                unique_id = hashlib.sha256(token.encode()).hexdigest()
                await self.async_set_unique_id(unique_id)
                self._abort_if_unique_id_configured()
                return self.async_create_entry(
                    title=user_input[CONF_NAME].strip(),
                    data={
                        CONF_NAME: user_input[CONF_NAME].strip(),
                        CONF_BRIDGE_ID: user_input[CONF_BRIDGE_ID].strip(),
                        CONF_PAIRING_TOKEN: token,
                        CONF_ALLOWED_LIST_IDS: allowed_list_ids,
                        CONF_ALLOWED_CALENDAR_IDS: _parse_allowed_list_ids(user_input.get(CONF_ALLOWED_CALENDAR_IDS, "")),
                    },
                )

        schema = vol.Schema(
            {
                vol.Optional(CONF_ALLOWED_CALENDAR_IDS, default=""): selector.TextSelector(
                    selector.TextSelectorConfig(multiline=True)
                ),
                vol.Required(CONF_NAME, default="iCloud Reminders Bridge"): str,
                vol.Required(CONF_BRIDGE_ID, default="mac"): str,
                vol.Required(CONF_PAIRING_TOKEN): selector.TextSelector(
                    selector.TextSelectorConfig(type=selector.TextSelectorType.PASSWORD)
                ),
                vol.Required(
                    CONF_ALLOWED_LIST_IDS,
                    default="",
                ): selector.TextSelector(
                    selector.TextSelectorConfig(multiline=True)
                ),
            }
        )
        return self.async_show_form(step_id="user", data_schema=schema, errors=errors)


def _parse_allowed_list_ids(value: str) -> list[str]:
    """Parse one exact EventKit list ID per line."""
    list_ids: list[str] = []
    seen: set[str] = set()
    for raw_id in value.splitlines():
        list_id = raw_id.strip()
        if list_id and list_id not in seen:
            list_ids.append(list_id)
            seen.add(list_id)
    return list_ids
