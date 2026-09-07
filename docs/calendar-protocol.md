# Calendar support

The native helper, MCP, and HA integration support read-only event calendars.
Real-account recurrence and background operation remain release validation work.

Protocol version 1 uses an additive `capabilities` declaration. The Mac bridge
advertises the scopes it is sending, and HA advertises the scopes it can return
in its response. Reminder-only clients remain interoperable with version-1
implementations that predate this field; a calendar-enabled client fails closed
when HA does not advertise `calendars`, rather than silently dropping events.
Unknown future capability names are ignored after basic shape validation.
The HA response also advertises `queue_epoch`; the Mac persists that value and
fences a changed or later-omitted epoch before applying queued mutations. See
[command lifecycle and recovery](command-lifecycle.md) for the explicit restore
review command.

For HA, install the updated custom component and restart HA before enabling
calendars in the Mac config. In integration setup, enter the same exact event
calendar IDs under Allowed EventKit calendar IDs. HA maintains its own allowlist;
older entries without calendar IDs reject incoming calendar snapshots. Existing
entries currently require setup again to add calendar IDs; take care to preserve
dashboard/entity references when replacing an entry.

HA publishes one read-only calendar entity per calendar received. The rolling
snapshot covers 30 days in the past and 90 days ahead, refreshed with each sync.
Queries outside the stored window return an explicit error. MCP callers can
request their own windows of up to 366 days. Calendar-only configurations can
use an empty reminder `lists` array. If a calendar read fails, the bridge still
publishes reminder changes and omits the `calendars` field for that snapshot.
HA retains the last confirmed calendar snapshot and exposes its last successful
calendar sync timestamp as `calendar_last_sync`; a later successful calendar
read replaces the cache. This keeps a Calendar permission or EventKit outage
from blocking Reminders while making the cached nature of the calendar view
visible to dashboards.

Existing HA entries can add or remove calendars through the integration's
Reconfigure action; entry identity, pairing, and reminder entities are retained.

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
