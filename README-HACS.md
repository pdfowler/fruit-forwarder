# Fruit Forwarder for Home Assistant

Fruit Forwarder connects Home Assistant to selected Apple Reminders lists and
Calendars through a local macOS EventKit bridge. The Mac application owns Apple
access; this repository contains only the Home Assistant integration.

This integration is distributed through HACS as a custom repository while it is
being validated for the default catalog. Install the matching Fruit Forwarder
Mac release first, configure its exact reminder and calendar IDs, then add this
integration and pair it with the generated token.

The integration exposes editable reminder todo lists and read-only calendar
entities. Calendar data is bounded to the bridge's published snapshot window.
The integration does not provide Apple account credentials, delete reminders,
edit calendar events, or discover data outside its configured allowlist.

See the main Fruit Forwarder documentation for Mac installation, MCP setup,
security boundaries, compatibility and troubleshooting.

- Source and cross-target documentation: [Fruit Forwarder](https://github.com/pdfowler/fruit-forwarder)
- HA release history: [releases](https://github.com/pdfowler/ha-fruit-forwarder/releases)
- Contributions and bug reports: use the [main repository issue tracker](https://github.com/pdfowler/fruit-forwarder/issues)
