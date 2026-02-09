#!/bin/bash
# Migration helper script

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR/.."

# Load .env if exists
if [ -f .env ]; then
    export $(cat .env | grep -v '^#' | xargs)
fi

case "$1" in
    up)
        echo "Running migrations up..."
        go run ./cmd/migrate -action=up
        ;;
    down)
        STEPS=${2:-1}
        echo "Rolling back $STEPS migration(s)..."
        go run ./cmd/migrate -action=down -steps=$STEPS
        ;;
    status)
        echo "Checking migration status..."
        go run ./cmd/migrate -action=status
        ;;
    create)
        if [ -z "$2" ]; then
            echo "Usage: $0 create <migration_name>"
            exit 1
        fi
        echo "Creating new migration: $2"
        go run ./cmd/migrate -action=create -name="$2"
        ;;
    reset)
        echo "Resetting database (down all + up all)..."
        go run ./cmd/migrate -action=down -steps=999
        go run ./cmd/migrate -action=up
        ;;
    *)
        echo "Usage: $0 {up|down [steps]|status|create <name>|reset}"
        echo ""
        echo "Commands:"
        echo "  up              - Apply all pending migrations"
        echo "  down [steps]    - Rollback migrations (default: 1)"
        echo "  status          - Show migration status"
        echo "  create <name>   - Create new migration files"
        echo "  reset           - Rollback all and reapply all migrations"
        exit 1
        ;;
esac

