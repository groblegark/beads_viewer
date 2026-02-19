#!/bin/sh
set -e

if [ -z "$BEADS_URL" ]; then
  echo "ERROR: BEADS_URL environment variable is required" >&2
  echo "  Set it to your bd daemon URL (e.g., http://bd-daemon:8443)" >&2
  exit 1
fi

echo "Loading issues from $BEADS_URL..."
bv --beads-url "$BEADS_URL" \
   ${BD_API_KEY:+--beads-api-key "$BD_API_KEY"} \
   --export-pages /app/pages \
   --robot

echo "Starting preview server on 0.0.0.0:${BV_PORT:-9000}..."
exec bv --preview-pages /app/pages \
        --preview-host 0.0.0.0 \
        --no-live-reload
