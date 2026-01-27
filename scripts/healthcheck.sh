#!/bin/sh
MAX_AGE=300 # 5 minutes

if [ -z "$HEALTHCHECK_FILE" ]; then
  echo "healthcheck disabled (HEALTHCHECK_FILE not set)"
  exit 0
fi

if [ ! -f "$HEALTHCHECK_FILE" ]; then
  echo "healthcheck file not found"
  exit 1
fi

# Use stat -c for GNU coreutils, stat -f for BusyBox/BSD
FILE_TIME=$(stat -c %Y "$HEALTHCHECK_FILE" 2>/dev/null || stat -f %m "$HEALTHCHECK_FILE" 2>/dev/null)
CURRENT_TIME=$(date +%s)
AGE=$((CURRENT_TIME - FILE_TIME))

if [ $AGE -gt $MAX_AGE ]; then
  echo "may be stalled (last update: ${AGE}s ago)"
  exit 1
fi

echo "healthy (last update: ${AGE}s ago)"
exit 0
