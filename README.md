## A DBA backup files tool set. ##

### Module BackupsControl is used to check the databases last backup files timestamps against a threshold. ###
It emails an operator about outdated backups for every group of backup files.

Files must obey naming scheme.
Database name and an optional suffix define a file group of related databases backups.
Every file group may have its most recent files and outdated files.
Uses Windows DPAPI to store email credentials.
Works with TLS enabled email server.
