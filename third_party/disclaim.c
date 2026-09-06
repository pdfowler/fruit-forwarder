/*
 * TCC responsibility shim, adapted from mcp-server-apple-events issue #93.
 * See LICENSE.mcp-server-apple-events in this directory.
 *
 * Usage: icloud-reminders-disclaim <binary> [args...]
 */
#include <spawn.h>
#include <stdio.h>
#include <string.h>

extern char **environ;

extern int responsibility_spawnattrs_setdisclaim(posix_spawnattr_t *attrs,
                                                  int disclaim)
    __attribute__((weak_import));

int main(int argc, char *argv[]) {
  if (argc < 2) {
    fprintf(stderr, "usage: %s <binary> [args...]\n",
            argc > 0 ? argv[0] : "icloud-reminders-disclaim");
    return 64;
  }

  posix_spawnattr_t attr;
  int rc = posix_spawnattr_init(&attr);
  if (rc != 0) {
    fprintf(stderr, "icloud-reminders-disclaim: posix_spawnattr_init: %s\n",
            strerror(rc));
    return 127;
  }
  rc = posix_spawnattr_setflags(&attr, POSIX_SPAWN_SETEXEC);
  if (rc != 0) {
    posix_spawnattr_destroy(&attr);
    fprintf(stderr, "icloud-reminders-disclaim: posix_spawnattr_setflags: %s\n",
            strerror(rc));
    return 127;
  }
  if (responsibility_spawnattrs_setdisclaim) {
    responsibility_spawnattrs_setdisclaim(&attr, 1);
  }

  pid_t pid;
  rc = posix_spawn(&pid, argv[1], NULL, &attr, &argv[1], environ);
  posix_spawnattr_destroy(&attr);
  fprintf(stderr, "icloud-reminders-disclaim: failed to exec %s: %s\n",
          argv[1], strerror(rc));
  return 127;
}
