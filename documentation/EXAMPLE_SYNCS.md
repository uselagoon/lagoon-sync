# The 'sync' command

Sync transfers are executed with `$lagoon-sync sync <syncer>` and require at least a syncer type `[mariadb|files|mongodb|postgres|drupalconfig]`, a valid project name `-p` and source environment `-e`. By default, if you do not provide an optional target environment `-t` then `local` is used.

```
lagoon-sync sync

Usage:
  lagoon-sync sync [mariadb|files|mongodb|postgres|etc.] [flags]

Flags:
  -c, --configuration-file string        File containing sync configuration.
      --dry-run                          Don't run the commands, just preview what will be run
  -h, --help                             help for sync
      --no-interaction                   Disallow interaction
  -p, --project-name string              The Lagoon project name of the remote system
  -r, --rsync-args string                Pass through arguments to change the behaviour of rsync (default "--omit-dir-times --no-perms --no-group --no-owner --chmod=ugo=rwX -r")
  -s, --service-name string              The service name (default is 'cli'
  -e, --source-environment-name string   The Lagoon environment name of the source system
  -i, --ssh-key string                   Specify path to a specific SSH key to use for authentication
  -t, --target-environment-name string   The target environment name (defaults to local)
      --verbose                          Run ssh commands in verbose (useful for debugging)

Global Flags:
      --config string   config file (default is .lagoon.yaml) (default "./.lagoon.yml")
      --show-debug      Shows debug information
```



## Example syncs

As with all sync commands, if you run into issues you can run `--show-debug` to see extra log information. There is also the `config` command which is useful to see what configuration files are active.

Note: All these examples mention your "current" environment. This can _either_ be your local development environment _or_ an environment deployed on Lagoon. "Current environment" here simply means "wherever you're running the lagoon-sync binary"

### Mariadb sync from remote source -> current environment
An example sync between a `mariadb` database from a remote source environment to your current environment may go as follows:

Running `$ lagoon-sync sync mariadb -p amazeelabsv4-com -e dev --dry-run` would dry-run a process that takes a database dump, runs a data transfer and then finally syncs the current environment's database with the latest dump.

### Mariadb sync from remote source to a file on your current environment (*Dump only*)

It's also possible to simply generate a backup from one of the remote servers by using the options
`--skip-target-cleanup=true`, which doesn't delete temporary transfer files, and `--skip-target-import=true` which
skips actually importing the database into your current environment.

`$ lagoon-sync sync mariadb -p amazeelabsv4-com -e prod --skip-target-cleanup=true --skip-target-import=true`

You will then see the transfer-resource name listed in the output.


