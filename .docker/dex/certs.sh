#!/usr/bin/env bash
# Generates the development certificates of the gRPC API of Dex: an authority, the
# server certificate of Dex, and the client certificate of the hub. Never use them
# outside a development machine.
set -euo pipefail

out="$(cd "$(dirname "$0")" && pwd)/certs"
days=825

mkdir -p "$out"
cd "$out"

openssl req -x509 -newkey ec -pkeyopt ec_paramgen_curve:P-256 -nodes \
  -keyout ca.key -out ca.crt -days "$days" -subj "/CN=maroid development authority"

openssl req -newkey ec -pkeyopt ec_paramgen_curve:P-256 -nodes \
  -keyout server.key -out server.csr -subj "/CN=dex"
openssl x509 -req -in server.csr -CA ca.crt -CAkey ca.key -CAcreateserial \
  -out server.crt -days "$days" \
  -extfile <(printf "subjectAltName=DNS:dex,DNS:localhost,IP:127.0.0.1\nextendedKeyUsage=serverAuth")

openssl req -newkey ec -pkeyopt ec_paramgen_curve:P-256 -nodes \
  -keyout client.key -out client.csr -subj "/CN=hub"
openssl x509 -req -in client.csr -CA ca.crt -CAkey ca.key -CAcreateserial \
  -out client.crt -days "$days" \
  -extfile <(printf "extendedKeyUsage=clientAuth")

rm -f server.csr client.csr ca.srl
chmod 600 ./*.key

echo "Wrote the certificates to $out"
