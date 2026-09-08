"""Config flow for Fruit Forwarder."""

from __future__ import annotations

import hashlib
import re
from typing import Any

import voluptuous as vol

from homeassistant import config_entries
from homeassistant.const import CONF_NAME
from homeassistant.helpers import selector

from .const import (
    CONF_ALLOWED_CALENDAR_IDS,
    CONF_ALLOWED_LIST_IDS,
    CONF_BRIDGE_ID,
    CONF_PAIRING_TOKEN,
    DOMAIN,
    MAX_LISTS,
    MAX_STRING_LENGTH,
)

TOKEN_PATTERN = re.compile(r"^[A-Za-z0-9_-]{43,128}$")


class ICloudRemindersBridgeConfigFlow(config_entries.ConfigFlow, domain=DOMAIN):
    """Configure a macOS EventKit reminders bridge."""

    VERSION = 1

    async def async_step_reconfigure(
        self, user_input: dict[str, Any] | None = None
    ) -> config_entries.ConfigFlowResult:
        """Change scope or rotate the pairing token in place."""
        entry = self._get_reconfigure_entry()
        errors: dict[str, str] = {}
        if user_input is not None:
            try:
                lists = _parse_allowed_list_ids(user_input.get(CONF_ALLOWED_LIST_IDS, ""))
                calendars = _parse_allowed_list_ids(user_input.get(CONF_ALLOWED_CALENDAR_IDS, ""))
            except ValueError:
                lists, calendars = [], []
                errors["base"] = "invalid_scope"
            token = user_input.get(CONF_PAIRING_TOKEN, "").strip()
            if errors:
                pass
            elif token and not TOKEN_PATTERN.fullmatch(token):
                errors[CONF_PAIRING_TOKEN] = "invalid_token"
            elif token and _token_in_use(self, token, entry):
                errors["base"] = "already_configured"
            elif not lists and not calendars:
                errors["base"] = "no_lists"
            else:
                data_updates = {
                    CONF_ALLOWED_LIST_IDS: lists,
                    CONF_ALLOWED_CALENDAR_IDS: calendars,
                }
                update_kwargs: dict[str, Any] = {"data_updates": data_updates}
                if token:
                    data_updates[CONF_PAIRING_TOKEN] = token
                    update_kwargs["unique_id"] = hashlib.sha256(token.encode()).hexdigest()
                return self.async_update_reload_and_abort(
                    entry,
                    **update_kwargs,
                )
        defaults = user_input if user_input is not None else {
            CONF_ALLOWED_LIST_IDS: "\n".join(entry.data.get(CONF_ALLOWED_LIST_IDS, [])),
            CONF_ALLOWED_CALENDAR_IDS: "\n".join(entry.data.get(CONF_ALLOWED_CALENDAR_IDS, [])),
            CONF_PAIRING_TOKEN: "",
        }
        return self.async_show_form(
            step_id="reconfigure", errors=errors,
            data_schema=vol.Schema(
                {
                    vol.Optional(CONF_PAIRING_TOKEN, default=""): selector.TextSelector(
                        selector.TextSelectorConfig(type=selector.TextSelectorType.PASSWORD)
                    ),
                    **{
                        vol.Required(key, default=defaults.get(key, "")): selector.TextSelector(
                            selector.TextSelectorConfig(multiline=True)
                        )
                        for key in (CONF_ALLOWED_LIST_IDS, CONF_ALLOWED_CALENDAR_IDS)
                    },
                }
            ),
        )

    async def async_step_user(
        self, user_input: dict[str, Any] | None = None
    ) -> config_entries.ConfigFlowResult:
        """Handle the initial setup form."""
        errors: dict[str, str] = {}
        if user_input is not None:
            name = user_input[CONF_NAME].strip()
            bridge_id = user_input[CONF_BRIDGE_ID].strip()
            token = user_input[CONF_PAIRING_TOKEN].strip()
            try:
                allowed_list_ids = _parse_allowed_list_ids(
                    user_input[CONF_ALLOWED_LIST_IDS]
                )
                allowed_calendar_ids = _parse_allowed_list_ids(
                    user_input.get(CONF_ALLOWED_CALENDAR_IDS, "")
                )
            except ValueError:
                allowed_list_ids, allowed_calendar_ids = [], []
                errors["base"] = "invalid_scope"
            if errors:
                pass
            elif not name or len(name) > MAX_STRING_LENGTH:
                errors[CONF_NAME] = "invalid_name"
            elif not bridge_id or len(bridge_id) > MAX_STRING_LENGTH:
                errors[CONF_BRIDGE_ID] = "invalid_bridge_id"
            elif _bridge_id_in_use(self, bridge_id):
                errors["base"] = "already_configured"
            elif not TOKEN_PATTERN.fullmatch(token):
                errors[CONF_PAIRING_TOKEN] = "invalid_token"
            elif not allowed_list_ids and not allowed_calendar_ids:
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
                        CONF_ALLOWED_CALENDAR_IDS: allowed_calendar_ids,
                    },
                )

        schema = vol.Schema(
            {
                vol.Optional(CONF_ALLOWED_CALENDAR_IDS, default=""): selector.TextSelector(
                    selector.TextSelectorConfig(multiline=True)
                ),
                vol.Required(CONF_NAME, default="Fruit Forwarder"): str,
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
    if not isinstance(value, str):
        raise ValueError("scope IDs must be text")
    list_ids: list[str] = []
    seen: set[str] = set()
    for raw_id in value.splitlines():
        list_id = raw_id.strip()
        if len(list_id) > MAX_STRING_LENGTH:
            raise ValueError("scope ID is too long")
        if list_id and list_id not in seen:
            list_ids.append(list_id)
            seen.add(list_id)
            if len(list_ids) > MAX_LISTS:
                raise ValueError("too many scope IDs")
    return list_ids


def _bridge_id_in_use(flow: ICloudRemindersBridgeConfigFlow, bridge_id: str) -> bool:
    """Reject a second config entry for the same native bridge identity."""
    if flow.hass is None:
        return False
    return any(
        entry.data.get(CONF_BRIDGE_ID) == bridge_id
        for entry in flow.hass.config_entries.async_entries(DOMAIN)
    )


def _token_in_use(
    flow: ICloudRemindersBridgeConfigFlow,
    token: str,
    current_entry: config_entries.ConfigEntry,
) -> bool:
    """Prevent two webhook registrations from sharing one bearer token."""
    if flow.hass is None:
        return False
    return any(
        entry is not current_entry
        and entry.data.get(CONF_PAIRING_TOKEN) == token
        for entry in flow.hass.config_entries.async_entries(DOMAIN)
    )
