"""Read-only calendars backed by bounded EventKit snapshots."""

from datetime import date

from homeassistant.components.calendar import CalendarEntity, CalendarEvent
from homeassistant.core import callback
from homeassistant.exceptions import HomeAssistantError
from homeassistant.util import dt as dt_util

from .calendar_data import instant


async def async_setup_entry(hass, entry, async_add_entities):
    runtime = entry.runtime_data
    entities = {}

    @callback
    def refresh():
        new = []
        for uid in runtime.calendars:
            if uid not in entities:
                entities[uid] = BridgeCalendar(runtime, uid)
                new.append(entities[uid])
        if new:
            async_add_entities(new)
        for entity in entities.values():
            entity._refresh_name()
            if entity.hass:
                entity.async_write_ha_state()

    entry.async_on_unload(runtime.subscribe(refresh))
    refresh()


class BridgeCalendar(CalendarEntity):
    _attr_has_entity_name = True
    # Poll only local freshness; event data still arrives through the bridge
    # snapshot and is never fetched by Home Assistant.
    _attr_should_poll = True
    _attr_supported_features = 0

    def __init__(self, runtime, uid):
        self.runtime = runtime
        self.uid = uid
        self._attr_unique_id = f"{runtime.entry.entry_id}:calendar:{uid}"
        self._attr_name = runtime.calendars[uid]["name"]

    @property
    def available(self):
        return self.uid in self.runtime.calendars and not self.runtime.snapshot_is_stale(
            calendar=True
        )

    @property
    def extra_state_attributes(self):
        calendar = self.runtime.calendars.get(self.uid, {})
        return {
            "window_start": calendar.get("window_start"),
            "window_end": calendar.get("window_end"),
            "last_sync": self.runtime.last_sync,
            "calendar_last_sync": self.runtime.calendar_last_sync,
            "sync_status": self.runtime.snapshot_status(calendar=True),
            "sync_age_seconds": self.runtime.snapshot_age_seconds(calendar=True),
        }

    async def async_update(self) -> None:
        """Refresh freshness locally even when the bridge is offline."""
        if self.hass:
            self.async_write_ha_state()

    def _events(self):
        result = []
        for item in self.runtime.calendars.get(self.uid, {}).get("events", []):
            parse = date.fromisoformat if item["all_day"] else instant
            result.append(CalendarEvent(start=parse(item["start"]), end=parse(item["end"]),
                summary=item["summary"], uid=item["uid"], description=item.get("description"),
                location=item.get("location")))
        return sorted(result, key=lambda event: event.start_datetime_local)

    @callback
    def _refresh_name(self) -> None:
        calendar = self.runtime.calendars.get(self.uid)
        if calendar is not None:
            self._attr_name = calendar["name"]

    @property
    def event(self):
        now = dt_util.now()
        return next((event for event in self._events() if event.end_datetime_local > now), None)

    async def async_get_events(self, hass, start_date, end_date):
        calendar = self.runtime.calendars.get(self.uid)
        if calendar is None:
            raise HomeAssistantError("Calendar snapshot is unavailable")
        if start_date < instant(calendar["window_start"]) or end_date > instant(calendar["window_end"]):
            raise HomeAssistantError("Requested dates exceed the bridge snapshot window (30 days past, 90 days ahead)")
        return [event for event in self._events()
                if event.start_datetime_local < end_date and event.end_datetime_local > start_date]
