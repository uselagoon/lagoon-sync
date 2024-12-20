# Custom synchers

It's possible to extend lagoon-sync to define your own sync processes. As lagoon-sync is essentially a
script runner that runs commands on target and source systems, as well as transferring data between the two systems,
it's possible to define commands that generate the transfer resource and consume it on the target.

For instance, if you have [mtk](https://github.com/skpr/mtk) set up on the target machine, it should be possible to
define a custom syncher that makes use of mtk to generate a sanitized DB dump on the source, and then use mysql to
import it on the target.

This is done by defining three things:
* The transfer resource name (what file is going to be synced across the network) - in this case let's call it "/tmp/dump.sql"
* The command(s) to run on the source
* The command(s) to run target

```
lagoon-sync:
  mtkdump:
    transfer-resource: "/tmp/dump.sql"
    source:
      commands:
        - "mtk-dump > {{ .transferResource }}"
    target:
      commands:
        - "mysql -h${MARIADB_HOST:-mariadb} -u${MARIADB_USERNAME:-drupal} -p${MARIADB_PASSWORD:-drupal} -P${MARIADB_PORT:-3306} ${MARIADB_DATABASE:-drupal} < {{ .transfer-resource }}"
```

This can then be called by running the following:
```
lagoon-sync sync mtkdump -p <SOURCE_PROJECT> -e <SOURCE_ENVIRONMENT>
```
