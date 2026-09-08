# Command lifecycle and recovery

Fruit Forwarder uses a durable, at-least-once command queue between Home
Assistant and the Mac EventKit process. The protocol deliberately does not
claim exactly-once EventKit execution: a process can stop after Apple accepts a
mutation but before the acknowledgement is durable.

## States

| State | Owner | Meaning | Automatic action |
| --- | --- | --- | --- |
| `queued` | Home Assistant | A user mutation is persisted locally and reflected optimistically in the HA entity | Include it in the next webhook response while its list is writable |
| `withheld` | Home Assistant | The target list is temporarily missing or read-only | Keep it persisted; do not send it to EventKit until a writable snapshot returns |
| `in_flight` | Mac bridge | The command is journaled immediately before the EventKit call | Stop all later command application until an operator resolves it |
| `confirmed` | Both | EventKit returned success and the Mac recorded the command ID in its acknowledgement ledger | Publish a fresh snapshot and let HA remove the matching queued command |
| `retryable_failure` | Mac/HA transport | No EventKit mutation was started, or a snapshot transport failed before command application | Retry using the normal poll interval or bounded transient-outage backoff |
| `uncertain` | Mac bridge | EventKit may have accepted the mutation, but the process cannot prove the outcome | Never blind-retry; inspect Apple Reminders and use explicit recovery |
| `permanent_failure` | HA or Mac validation | The request is out of scope, malformed, unsupported, or targets an unavailable item | Reject it and surface the actionable error; do not keep replaying it |

HA exposes queued work through the todo entity's `pending_commands` attribute
and optimistic item state. A successful Mac acknowledgement removes the command
on the next confirmed snapshot. A bridge outage leaves the last confirmed
snapshot visible; it must not be presented as newly synchronized.

## Snapshot ordering

Native bridge snapshots include an ISO-8601 `sent_at` timestamp. After the HA
entry accepts its first timestamped snapshot, it rejects older or duplicate
snapshots and untimestamped downgrades, while allowing at most five minutes of
future clock skew. This prevents a delayed valid webhook replay from replacing
newer confirmed state. Older protocol peers may send one initial untimestamped
snapshot, but current bridge releases always send the timestamp.

## Ambiguous EventKit outcomes

The Mac persists the complete command as `in_flight` before invoking EventKit.
If the helper returns an error, times out, or the process is interrupted after
that journal write, synchronization pauses. `doctor --json` reports the command
ID. After checking Apple Reminders manually, choose exactly one resolution:

```sh
bridge recover --command-id COMMAND_ID --resolution applied
# or, only when Apple did not change the item:
bridge recover --command-id COMMAND_ID --resolution retry
```

`applied` records the command as acknowledged without invoking EventKit again.
`retry` clears the journal so the command may be sent on a later sync. Neither
choice proves an external Apple edit; the operator's inspection is part of the
recovery evidence. Both recovery resolutions acquire the same state lock as
normal synchronization, so they cannot race a running `serve` or `sync-once`.

## Retention and bounds

The Mac acknowledgement ledger retains at most 1,000 command IDs, while HA
bounds queued commands and payload sizes. Eviction never removes a pending or
in-flight command. HA also persists a queue epoch beside its pending commands,
and the Mac persists the epoch it has accepted. If a restored HA backup presents
a different epoch, the Mac refuses to apply any queued command until an operator
reviews the backup and explicitly accepts it:

```sh
bridge reset-queue-epoch --queue-epoch EPOCH_FROM_THE_REVIEWED_HA_QUEUE
```

Restored state is validated before use; malformed, oversized, duplicate, or
blank entries fail closed. A peer that omits the epoch remains compatible only
until the Mac has established one; later omission is treated as an unsafe
downgrade. These bounds and the epoch fence protect against common stale-backup
replays but are not a substitute for a durable backup or human review.

## Conflict boundaries

Commands are scoped to exact EventKit list and item identifiers. An update is
validated against the current list snapshot, and a missing item is rejected
rather than silently applied to a same-named reminder. Concurrent edits made
outside the bridge can still race with a queued command; an ambiguous result
must use the recovery procedure above. Calendar data is read-only and never
enters the mutation queue.
