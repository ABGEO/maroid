#!/bin/sh
# Creates the bucket of the hub, its CORS rule, and a key that reaches that bucket alone.
set -eu

rc alias set store http://rustfs:9000 "$ROOT_ACCESS_KEY" "$ROOT_SECRET_KEY"
rc bucket create store/maroid
rc bucket cors set store/maroid /setup/cors.xml
rc admin user add store "$HUB_ACCESS_KEY" "$HUB_SECRET_KEY"
rc admin policy create store maroid-hub /setup/policy.json
rc admin policy attach store maroid-hub --user "$HUB_ACCESS_KEY"
