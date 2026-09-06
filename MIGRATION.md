# Moving from an embedded deployment

This repository is the standalone distribution of the bridge that was
previously kept inside another infrastructure repository. The Home Assistant
custom component, EventKit helper, Go service, and deployment script are now
versioned together here.

The standalone defaults deliberately use generic paths and the placeholder
reverse-DNS identifier `com.example.icloud-reminders-bridge`. Existing users
should keep their current runtime config and Keychain item until the new build
has been tested, then update the LaunchAgent and Home Assistant integration as
a coordinated change. Never copy a runtime config, state file, or pairing
token into this repository.
