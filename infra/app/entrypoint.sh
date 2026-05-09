#!/bin/sh
set -e

# Fix ownership of bind-mounted directories so appuser can write to them
chown -R appuser:appgroup /app/db

# Drop privileges and exec the application
exec gosu appuser "$@"
