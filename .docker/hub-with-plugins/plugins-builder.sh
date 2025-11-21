#!/usr/bin/env sh

set -e

PLUGINS_ROOT=${PLUGINS_ROOT:-"/plugins"}
OUT_DIR=${OUT_DIR:-"/plugin-out"}

echo "==> M a r o i d   P l u g i n   B u i l d e r"
echo "Scanning for plugins in: ${PLUGINS_ROOT}"
echo

PLUGIN_DIRS=$(find "$PLUGINS_ROOT" -mindepth 1 -maxdepth 1 -type d | sort)
if [ -z "$PLUGIN_DIRS" ]; then
  echo "ERROR: No plugins found in ${PLUGINS_ROOT}"

  exit 1
fi

echo "Found plugins: $(echo "$PLUGIN_DIRS" | xargs -n1 basename | tr '\n' ' ')"

for PLUGIN_PATH in $PLUGIN_DIRS; do
  PLUGIN_NAME=$(basename "$PLUGIN_PATH")
  PLUGIN_SOURCE_PATH="${PLUGIN_PATH}" PLUGIN_OUT_DIR="${OUT_DIR}" /build.sh "${PLUGIN_NAME}"
done

echo "==============================================="
echo "ALL PLUGINS BUILT SUCCESSFULLY"
echo "Output directory: ${OUT_DIR}"
echo "==============================================="
