#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
JAR_FILE="$SCRIPT_DIR/karate.jar"
JAR_URL="https://github.com/karatelabs/karate/releases/download/v1.4.1/karate-1.4.1.jar"
echo "Cleaning up previous test-generated grade items from database..."
if command -v docker >/dev/null 2>&1; then
  docker exec -i ejournal-postgres psql -U postgres -d ejournal -c "DELETE FROM grade_items WHERE title LIKE 'Independent Lab%' OR title LIKE 'Lab Assignment%';" >/dev/null 2>&1 || true
fi
if command -v kubectl >/dev/null 2>&1; then
  kubectl exec -n ejournal postgres-0 -- psql -U postgres -d ejournal -c "DELETE FROM grade_items WHERE title LIKE 'Independent Lab%' OR title LIKE 'Lab Assignment%';" >/dev/null 2>&1 || true
fi

if [ ! -f "$JAR_FILE" ]; then
  echo "Karate Standalone JAR not found. Downloading v1.4.1..."
  curl -L "$JAR_URL" -o "$JAR_FILE"
  echo "Download completed."
fi

java -Dkarate.config.dir="$SCRIPT_DIR" -jar "$JAR_FILE" "$@"
