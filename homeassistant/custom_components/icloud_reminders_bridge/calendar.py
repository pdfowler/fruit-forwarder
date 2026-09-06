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
            if entity.hass:
                entity.async_write_ha_state()

    entry.async_on_unload(runtime.subscribe(refresh))
    refresh()


class BridgeCalendar(CalendarEntity):
    _attr_has_entity_name = True
    _attr_should_poll = False
    _attr_supported_features = 0

    def __init__(self, runtime, uid):
        self.runtime = runtime
        self.uid = uid
        self._attr_unique_id = f"{runtime.entry.entry_id}:calendar:{uid}"
        self._attr_name = runtime.calendars[uid]["name"]

    @property
    def available(self):
        return self.uid in self.runtime.calendars

    @property
    def extra_state_attributes(self):
        calendar = self.runtime.calendars.get(self.uid, {})
        return {"window_start": calendar.get("window_start"),
                "window_end": calendar.get("window_end"), "last_sync": self.runtime.last_sync}

    def _events(self):
        result = []
        for item in self.runtime.calendars.get(self.uid, {}).get("events", []):
            parse = date.fromisoformat if item["all_day"] else instant
            result.append(CalendarEvent(start=parse(item["start"]), end=parse(item["end"]),
                summary=item["summary"], uid=item["uid"], description=item.get("description"),
                location=item.get("location")))
        return sorted(result, key=lambda event: event.start_datetime_local)

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
