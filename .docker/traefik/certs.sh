#!/usr/bin/env bash
# Generates the development certificate of Traefik for maroid.localhost and its
# subdomains. mkcert signs it with a local authority that the browser trusts.
set -euo pipefail

out="$(cd "$(dirname "$0")" && pwd)/certs"

mkdir -p "$out"
mkcert -install
mkcert -cert-file "$out/maroid-localhost.pem" -key-file "$out/maroid-localhost-key.pem" \
  maroid.localhost '*.maroid.localhost'
