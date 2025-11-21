#!/usr/bin/env sh

set -e

PLUGIN_NAME=$1

PLUGIN_SOURCE_PATH=${PLUGIN_SOURCE_PATH:-"/src"}
PLUGIN_OUT_DIR=${PLUGIN_OUT_DIR:-"/plugin-out"}

if [ -z "$PLUGIN_NAME" ] ; then
  echo "ERROR: No plugin name provided."

  exit 1
fi

echo "-----------------------------------------------"
echo "Compiling Maroid plugin: $PLUGIN_NAME"
echo
echo "PLUGIN_SOURCE_PATH: ${PLUGIN_SOURCE_PATH}"
echo "PLUGIN_OUT_DIR: ${PLUGIN_OUT_DIR}"
echo "-----------------------------------------------"

if [ ! -f "$PLUGIN_SOURCE_PATH/go.mod" ]; then
  echo "ERROR: Missing go.mod in plugin directory: $PLUGIN_SOURCE_PATH"
  echo "Every plugin must be a standalone Go module."

  exit 1
fi

mkdir -p "$PLUGIN_OUT_DIR"
cd "$PLUGIN_SOURCE_PATH"

echo "-> Running go mod tidy..."
go mod tidy

echo "-> Compiling ${PLUGIN_NAME}.so..."
CGO_ENABLED=1 go build \
  -v \
  -ldflags="-s -w" \
  -trimpath \
  -buildmode=plugin \
  -o "${PLUGIN_OUT_DIR}/${PLUGIN_NAME}.so" \
  "."

echo "✓ Built: ${PLUGIN_OUT_DIR}/${PLUGIN_NAME}.so"
