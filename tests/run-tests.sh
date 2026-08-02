#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
JAR_FILE="$SCRIPT_DIR/karate.jar"
JAR_URL="https://github.com/karatelabs/karate/releases/download/v1.4.1/karate-1.4.1.jar"

if [ ! -f "$JAR_FILE" ]; then
  echo "Karate Standalone JAR not found. Downloading v1.4.1..."
  curl -L "$JAR_URL" -o "$JAR_FILE"
  echo "Download completed."
fi

java -Dkarate.config.dir="$SCRIPT_DIR" -jar "$JAR_FILE" "$@"
