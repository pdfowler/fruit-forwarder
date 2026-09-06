# Contributing

Contributions are welcome. Please keep changes narrow and avoid adding
credentials, reminder contents, EventKit identifiers, or generated build
artifacts.

Before opening a pull request on macOS, run:

```sh
go test -race ./...
go vet ./...
bash -n scripts/install-macos.sh
```

The native helper needs macOS and a user-granted Reminders permission. Changes
to the Home Assistant component should preserve the list allowlist and the
bounded completed-item retention behavior.
