#!/usr/bin/env bash
set -euo pipefail

MSSQL_HOST="${MSSQL_HOST:-mssql}"
MSSQL_PORT="${MSSQL_PORT:-1433}"
MSSQL_USER="${MSSQL_USER:-sa}"
MSSQL_DATABASE="${MSSQL_DATABASE:-master}"

echo "Waiting for SQL Server at ${MSSQL_HOST}:${MSSQL_PORT}..."

for i in {1..60}; do
  if /opt/mssql-tools18/bin/sqlcmd \
    -S "${MSSQL_HOST},${MSSQL_PORT}" \
    -U "${MSSQL_USER}" \
    -P "${MSSQL_SA_PASSWORD}" \
    -d "${MSSQL_DATABASE}" \
    -C \
    -Q "SELECT 1" \
    -l 1 >/dev/null 2>&1; then
    echo "SQL Server is ready."
    break
  fi

  if [ "$i" -eq 60 ]; then
    echo "SQL Server did not become ready in time."
    exit 1
  fi

  sleep 2
done

run() {
  local file="$1"

  echo "Running ${file}"

  /opt/mssql-tools18/bin/sqlcmd \
    -S "${MSSQL_HOST},${MSSQL_PORT}" \
    -U "${MSSQL_USER}" \
    -P "${MSSQL_SA_PASSWORD}" \
    -d "${MSSQL_DATABASE}" \
    -C \
    -i "${file}"
}

run /initdb/01-create-db-and-schema.sql
run /initdb/02a-create-tables.sql
run /initdb/02b-constraints-and-indexes.sql

if [ -f /initdb/02c-inline-fk-patch.sql ]; then
  run /initdb/02c-inline-fk-patch.sql
fi

run /initdb/03-enable-cdc.sql

echo "All initialization scripts completed."
