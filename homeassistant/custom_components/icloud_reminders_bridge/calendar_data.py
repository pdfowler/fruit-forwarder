"""Validation shared by stored and incoming calendar snapshots."""

from datetime import date, datetime, timedelta

from .const import MAX_STRING_LENGTH


def instant(value):
    if not isinstance(value, str) or len(value) > MAX_STRING_LENGTH:
        raise ValueError("Calendar timestamp must be a string")
    parsed = datetime.fromisoformat(value)
    if parsed.tzinfo is None:
        raise ValueError("Calendar timestamp must include a timezone")
    return parsed


def validate_calendars(raw, allowed):
    if not isinstance(raw, list) or len(raw) > 100:
        raise ValueError("Invalid calendar collection")
    result = {}
    count = 0
    for calendar in raw:
        if not isinstance(calendar, dict):
            raise ValueError("Calendar must be an object")
        uid = calendar.get("id")
        if (not isinstance(uid, str) or len(uid) > MAX_STRING_LENGTH
                or uid not in allowed or uid in result):
            raise ValueError("Unexpected or duplicate calendar identifier")
        if (not isinstance(calendar.get("name"), str)
                or len(calendar["name"]) > MAX_STRING_LENGTH
                or not calendar["name"].strip()):
            raise ValueError("Calendar name must be non-empty")
        start, end = instant(calendar.get("window_start")), instant(calendar.get("window_end"))
        if not start < end or end - start > timedelta(days=366):
            raise ValueError("Invalid calendar window")
        events = calendar.get("events")
        if not isinstance(events, list):
            raise ValueError("Events must be an array")
        count += len(events)
        if count > 10000:
            raise ValueError("Too many calendar events")
        seen = set()
        clean = []
        for event in events:
            if not isinstance(event, dict):
                raise ValueError("Event must be an object")
            eid = event.get("uid")
            if not isinstance(eid, str) or not eid or len(eid) > MAX_STRING_LENGTH or eid in seen:
                raise ValueError("Invalid or duplicate event identifier")
            seen.add(eid)
            if not isinstance(event.get("all_day"), bool):
                raise ValueError("all_day must be boolean")
            for key in ("summary", "start", "end"):
                if (not isinstance(event.get(key), str)
                        or len(event[key]) > MAX_STRING_LENGTH):
                    raise ValueError("Event fields must be strings")
            for key in ("description", "location", "time_zone"):
                if (key in event and (not isinstance(event[key], str)
                        or len(event[key]) > MAX_STRING_LENGTH)):
                    raise ValueError("Event metadata must be strings")
            parse = date.fromisoformat if event["all_day"] else instant
            a, b = parse(event["start"]), parse(event["end"])
            if b <= a:
                raise ValueError("Invalid event interval")
            clean.append({key: event[key] for key in (
                "uid", "summary", "start", "end", "all_day", "description", "location", "time_zone"
            ) if key in event})
        result[uid] = {"id": uid, "name": calendar["name"], "window_start": calendar["window_start"],
                       "window_end": calendar["window_end"], "events": clean}
    return result
