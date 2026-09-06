# Calendar adapter (under development)

The native helper and MCP support read-only event calendars. HA calendar entities
are not wired yet; calendar configuration currently affects MCP only.

After installing the updated binaries, run `icloud-reminders-bridge discover-calendars`
from an interactive terminal and grant Calendar permission. Copy exact IDs into
a separate `calendars` array in the bridge config, for example:

```json
"calendars": [{"id": "EXACT-CALENDAR-ID", "name": "Personal"}]
```

Restart the MCP session. `calendar_lists` returns configured calendars, and
`calendar_events` accepts `calendar_id`, `start`, and `end`. Both tools are reads
and remain available in `mcp_read_only` mode. Omitting `calendars` disables both
tools and avoids requesting Calendar access. Discovery is a local CLI action;
MCP cannot discover calendars outside the configured scope.

`calendars` discovers event-calendar IDs, names, and sources. It requests Calendar
full access separately from Reminders. Apple's full-access permission is broader
than the application's allowlist; see [Apple's EventKit access guidance](https://developer.apple.com/documentation/eventkit/accessing-the-event-store).

`calendar_snapshot` accepts explicit `calendar_ids`, `start`, and `end` fields:

```json
{
  "action": "calendar_snapshot",
  "calendar_ids": ["EXACT-CALENDAR-ID"],
  "start": "2026-01-01T00:00:00Z",
  "end": "2026-02-01T00:00:00Z"
}
```

An empty ID array returns no calendars and does not request access. IDs must
be unique, non-empty, and at most 100 in number. The positive RFC3339 time window
is capped at 366 days. Snapshots exceeding 10,000 returned events fail rather
than silently truncate. Fetching uses EventKit's scoped event predicate.

Responses contain `calendars`, each with `id`, `name`, `source`, and `events`.
Events contain `uid`, `summary`, `start`, `end`, `all_day`, `time_zone`,
`description`, and `location`. All-day dates use YYYY-MM-DD with exclusive end
dates; timed events use UTC RFC3339 with the source time-zone identifier also
provided. Occurrence IDs combine the EventKit calendar-item ID and start instant;
moving an occurrence changes that identifier. Calendar writes are not exposed.

After building the native helper, run `python3 scripts/test-native-calendar.py`
to check empty-scope and invalid-request behavior without reading account data.
Real recurring-event and daylight-saving behavior still needs fixture validation.
