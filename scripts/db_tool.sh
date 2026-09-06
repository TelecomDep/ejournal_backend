#!/usr/bin/env bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PYTHON_SCRIPT="$SCRIPT_DIR/backup_manager.py"

case "$1" in
    backup)
        python3 "$PYTHON_SCRIPT" --backup
        ;;
    validate)
        shift
        python3 "$PYTHON_SCRIPT" --validate "$@"
        ;;
    wipe)
        shift
        python3 "$PYTHON_SCRIPT" --wipe "$@"
        ;;
    backup-and-wipe)
        shift
        python3 "$PYTHON_SCRIPT" --backup-and-wipe "$@"
        ;;
    restore)
        shift
        python3 "$PYTHON_SCRIPT" --restore "$@"
        ;;
    restore-full)
        shift
        python3 "$PYTHON_SCRIPT" --restore-full "$@"
        ;;
    *)
        echo "Использование: $0 {backup|validate|wipe|backup-and-wipe|restore|restore-full}"
        exit 1
        ;;
esac
