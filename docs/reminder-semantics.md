# Reminder field semantics

Fruit Forwarder exposes a deliberately small reminder model. EventKit remains
the source of truth; the bridge does not attempt to recreate every Apple
Reminders feature in Home Assistant or MCP.

| Field or feature | Read | Create | Update | Behavior |
| --- | --- | --- | --- | --- |
| Stable EventKit UID | Yes | No, assigned by EventKit | Required target | Exact list and item scope; display-name changes do not change identity |
| Title | Yes | Required | Required | Whitespace-only titles are rejected |
| Notes/description | Yes | Optional | HA full replacement; MCP patch | MCP omission preserves the existing value; explicit empty string clears it |
| Due date/time | Yes | Optional | HA full replacement; MCP patch | Supports `YYYY-MM-DD` or RFC3339; MCP omission preserves, explicit empty clears |
| Completion status | Yes | Optional | Required for update | `needs_action` or `completed`; complete/reopen are separate operations |
| Completion timestamp | Read-only | No | No | Supplied by EventKit after completion |
| Last-modified timestamp | Read-only | No | No | Supplied by EventKit |
| Recurrence, subtasks, attachments, tags, priority, location, alarms | Not represented | No | No | Not exposed as bridge fields; updating supported fields leaves these EventKit-owned fields untouched |
| Delete/list management | No | No | No | Intentionally unsupported |

Home Assistant sends the current todo item when queuing an update, so its
description and due fields use full-replacement semantics. MCP callers should
omit optional patch fields unless they intend to preserve or clear them as
documented above. The MCP server never interprets reminder text as commands.

Completed items are filtered from native snapshots according to
`completed_retention` (30 days by default), so the bridge does not continually
import an unbounded historical archive into Home Assistant or model context.
