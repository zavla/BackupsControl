### Module BackupsControl is used to check the databases backup files (last ones) timestamps.

It emails an operator in case of absence or outdated backup for every group of backup files.

Backup files divided in groups by name template in config file config.json.
Every database can have two or more groups of files: one for the full backups and one for the differential backups for example.
Uses Windows DPAPI to store email credentials.
Works with TLS enabled email server.
