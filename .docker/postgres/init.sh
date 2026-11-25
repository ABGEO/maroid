#!/bin/sh
# The hub and Dex connect as roles without the superuser and the BYPASSRLS
# attributes, because PostgreSQL never applies a row level security policy to a
# superuser. The entrypoint runs this file one time, when the cluster initializes.
set -eu

psql -v ON_ERROR_STOP=1 \
    --username "${POSTGRES_USER}" \
    --dbname "${POSTGRES_DB}" \
    --set user="${MAROID_DB_USER}" \
    --set password="${MAROID_DB_PASSWORD}" \
    --set database="${MAROID_DB_NAME}" \
    --set dex_user="${DEX_DB_USER}" \
    --set dex_password="${DEX_DB_PASSWORD}" \
    --set dex_database="${DEX_DB_NAME}" <<'SQL'
CREATE ROLE :"user" LOGIN PASSWORD :'password' NOSUPERUSER NOCREATEROLE NOBYPASSRLS;
CREATE ROLE :"dex_user" LOGIN PASSWORD :'dex_password' NOSUPERUSER NOCREATEROLE NOBYPASSRLS;

CREATE DATABASE :"database" OWNER :"user";
CREATE DATABASE :"dex_database" OWNER :"dex_user";

\connect :"database"

-- CREATE EXTENSION timescaledb needs a superuser, and the core migration runs as
-- the role above. Its IF NOT EXISTS clause finds the extension and skips it.
CREATE EXTENSION IF NOT EXISTS timescaledb;
SQL
