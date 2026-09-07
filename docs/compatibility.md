# Compatibility matrix

This is the pre-release support statement. A green build proves compilation
and synthetic boundaries; it does not grant Apple permissions or prove a
background launch. Update this table only with recorded acceptance evidence.

| Component | Verified baseline | Current claim |
| --- | --- | --- |
| macOS native bridge | macOS runner, Apple-silicon build, Swift EventKit boundary tests | macOS required; foreground EventKit path is implemented; background/TCC/reboot support remains under validation |
| Home Assistant | HA 2026.2.3, Python 3.13 runtime tests | Custom/HACS-shaped integration; live HA install and full entity lifecycle remain release gates |
| MCP | Local stdio subprocess tests; MCPB manifest and checksum validation | macOS MCPB candidate; individual client installation and permission behavior require acceptance testing |
| Protocol | Version 1 | Mac bridge and HA integration must use matching protocol version; incompatible payloads fail closed |

Supported scope is scoped Apple Reminders read/create/update/complete/reopen and
read-only EventKit calendars. Network MCP, Calendar writes, reminder deletion,
list management, Find My, and other iCloud services are not supported claims.
