# Moving from an embedded deployment

This repository is the standalone distribution of the bridge that was
previously kept inside another infrastructure repository. The Home Assistant
custom component, EventKit helper, Go service, and deployment script are now
versioned together here.

The standalone distribution uses the maintainer-owned reverse-DNS identity
`com.pdfowler.fruitforwarder` (with the EventKit helper identity
`com.pdfowler.fruitforwarder.eventkit`). Existing users of the prototype
identity `com.example.icloud-reminders-bridge` should keep their current
runtime config and Keychain item while testing the new build. The installer
stops the legacy LaunchAgent before activating the stable one; an existing
config that explicitly names the legacy Keychain service remains usable.
Never copy a runtime config, state file, or pairing token into this repository.
