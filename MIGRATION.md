# Moving from an embedded deployment

This repository is the standalone distribution of the bridge that was
previously kept inside another infrastructure repository. The Home Assistant
custom component, EventKit helper, Go service, and deployment script are now
versioned together here.

The standalone distribution uses the maintainer-owned reverse-DNS identity
`com.pdfowler.fruitforwarder` (with the EventKit helper identity
`com.pdfowler.fruitforwarder.eventkit`). Existing users of the former
`home-ctrl` deployment (`net.pdfowler.icloud-reminders-bridge`) or the earlier
prototype identity (`com.example.icloud-reminders-bridge`) should keep their
current runtime config and Keychain item while testing the new build. The
installer requires the explicit `--migrate-home-ctrl` option before copying a
home-ctrl config, then stops those legacy LaunchAgents when that migration is
activated. The migration copies the old config into the standalone path and preserves
its explicit legacy Keychain and state paths, so the existing pairing and
acknowledgement ledger remain available. Review the copied config before
activation.

Never copy a pairing token into this repository. Runtime config and state stay
on the Mac.

For a packaged migration on Dean:

```sh
scripts/install-package-macos.sh --migrate-home-ctrl --install-only
# Review ~/.config/icloud-reminders-bridge/config.json, then activate:
scripts/install-package-macos.sh --migrate-home-ctrl
```
