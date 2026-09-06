"""Constants for the iCloud Reminders Bridge integration."""

from homeassistant.const import Platform

DOMAIN = "icloud_reminders_bridge"
PLATFORMS = [Platform.TODO, Platform.CALENDAR]
PROTOCOL_VERSION = 1

CONF_BRIDGE_ID = "bridge_id"
CONF_PAIRING_TOKEN = "pairing_token"
CONF_ALLOWED_LIST_IDS = "allowed_list_ids"
CONF_ALLOWED_CALENDAR_IDS = "allowed_calendar_ids"

MAX_PAYLOAD_BYTES = 1024 * 1024
MAX_LISTS = 100
MAX_ITEMS = 10000
