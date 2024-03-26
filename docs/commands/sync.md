# sync

Sync transfers are executed with `$ lagoon-sync sync <syncer>` and require at least a syncer type `[mariadb|files|mongodb|postgres|drupalconfig]`, a valid project name `-p` and source environment `-e`. By default, if you do not provide an optional target environment `-t` then `local` is used.

```
lagoon-sync sync

Usage:
  lagoon-sync sync [mariadb|files|mongodb|postgres|etc.] [flags]

Flags:
      --dry-run                          Don't run the commands, just preview what will be run
  -h, --help                             help for sync
      --no-interaction                   Disallow interaction
  -p, --project-name string              The Lagoon project name of the remote system
  -r, --rsync-args string                Pass through arguments to change the behaviour of rsync (default "--omit-dir-times --no-perms --no-group --no-owner --chmod=ugo=rwX --recursive --compress")
  -s, --service-name string              The service name (default is 'cli'
      --skip-source-cleanup              Don't clean up any of the files generated on the source
      --skip-target-cleanup              Don't clean up any of the files generated on the target
      --skip-target-import               This will skip the import step on the target, in combination with 'no-target-cleanup' this essentially produces a resource dump
  -e, --source-environment-name string   The Lagoon environment name of the source system
  -H, --ssh-host string                  Specify your lagoon ssh host, defaults to 'ssh.lagoon.amazeeio.cloud' (default "ssh.lagoon.amazeeio.cloud")
  -i, --ssh-key string                   Specify path to a specific SSH key to use for authentication
  -P, --ssh-port string                  Specify your ssh port, defaults to '32222' (default "32222")
  -t, --target-environment-name string   The target environment name (defaults to local)
      --verbose                          Run ssh commands in verbose (useful for debugging)

Global Flags:
      --config string   config file (default is .lagoon.yaml) (default "./.lagoon.yml")
      --show-debug      Shows debug information
```