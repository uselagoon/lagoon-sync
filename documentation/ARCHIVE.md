# Archive and Extract

The `archive` and `extract` commands let you snapshot an environment's data into a portable file and restore it elsewhere. This is useful for taking local backups, migrating data between environments that can't reach each other directly, or seeding a local development environment from a production snapshot.

## `archive`

Dumps the databases and file volumes discovered in your `docker-compose.yml` into a single `.tar.gz` archive.

```
lagoon-sync archive [flags]
```

The command reads your `docker-compose.yml`, identifies MariaDB, PostgreSQL, and file-volume services, and packages them up:

- MariaDB and PostgreSQL databases are dumped to compressed `.sql.gz` files inside the archive.
- File volumes are included as-is.
- A `manifest.yml` is embedded in the archive so `extract` knows how to restore everything.

**Flags**

| Flag | Default | Description |
|------|---------|-------------|
| `-f, --docker-compose-file` | `docker-compose.yml` | Path to the docker-compose file to read services from. |
| `--archive-output` | `archive.tar.gz` | Filename for the generated archive. Must end in `.tar.gz`. |
| `--override-volume` | _(none)_ | Explicitly specify a volume path to include instead of auto-discovering file volumes. Repeatable. |
| `--use-service-api` | `false` | Use the Lagoon service API to discover services instead of docker-compose. |
| `-H, --ssh-host` | `ssh.lagoon.amazeeio.cloud` | SSH host for Lagoon. |
| `-P, --ssh-port` | `32222` | SSH port for Lagoon. |
| `-i, --ssh-key` | _(none)_ | Path to a specific SSH key to use for authentication. |
| `-A, --api` | `https://api.lagoon.amazeeio.cloud/graphql` | Lagoon API endpoint (used with SSH portal integration). |

**Example — archive using defaults**

Run this inside your `cli` container (or wherever `lagoon-sync` is available):

```sh
lagoon-sync archive
```

This reads `docker-compose.yml` in the current directory, dumps all discovered databases, and writes `archive.tar.gz`.

**Example — custom output path and docker-compose file**

```sh
lagoon-sync archive -f /app/docker-compose.yml --archive-output /tmp/my-snapshot.tar.gz
```

**Example — override the file volumes captured**

If you want to capture specific paths rather than relying on auto-discovery:

```sh
lagoon-sync archive --override-volume /app/web/sites/default/files --override-volume /app/private
```

---

## `extract`

Restores an archive created by `lagoon-sync archive` into the local environment.

```
lagoon-sync extract --archive-input <file> [flags]
```

The command reads the manifest from the archive and replays each item:

- MariaDB and PostgreSQL dumps are imported into the running database services.
- Files are extracted to the `--extraction-root` (defaults to `/`, preserving the original paths).

`--archive-input` is required.

**Flags**

| Flag | Default | Description |
|------|---------|-------------|
| `--archive-input` | _(required)_ | Path to the `.tar.gz` archive to restore. |
| `--extraction-root` | `/` | Root path used when extracting file items. Useful when restoring into a different directory layout. |
| `--dry-run` | `false` | Print the commands that would be run without executing them. |
| `--use-service-api` | `false` | Use the Lagoon service API. |
| `-H, --ssh-host` | `ssh.lagoon.amazeeio.cloud` | SSH host for Lagoon. |
| `-P, --ssh-port` | `32222` | SSH port for Lagoon. |
| `-i, --ssh-key` | _(none)_ | Path to a specific SSH key to use for authentication. |
| `-A, --api` | `https://api.lagoon.amazeeio.cloud/graphql` | Lagoon API endpoint. |

**Example — restore an archive**

```sh
lagoon-sync extract --archive-input archive.tar.gz
```

**Example — dry run to preview what will be restored**

```sh
lagoon-sync extract --archive-input archive.tar.gz --dry-run
```

**Example — extract files to a specific root**

```sh
lagoon-sync extract --archive-input archive.tar.gz --extraction-root /app
```

---

## Typical workflow

1. On the **source** environment (e.g. production), create an archive:

   ```sh
   lagoon-sync archive --archive-output snapshot.tar.gz
   ```

2. Copy `snapshot.tar.gz` to the **target** environment (e.g. local or staging).

3. On the **target** environment, restore it:

   ```sh
   lagoon-sync extract --archive-input snapshot.tar.gz
   ```

Both commands need to run inside a container where `lagoon-sync` is installed — the same requirement as the `sync` commands. See [Installation](./INSTALLATION.md) for how to include `lagoon-sync` in your container image.
