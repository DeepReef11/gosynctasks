#!/bin/bash
# Stop script for Nextcloud test server

set -e

echo "=== Stopping Nextcloud Test Server ==="
echo ""

# Check if docker compose is available
if docker compose version &> /dev/null; then
    COMPOSE_CMD="docker compose"
elif command -v docker-compose &> /dev/null; then
    COMPOSE_CMD="docker-compose"
else
    echo "ERROR: Docker Compose is not installed"
    exit 1
fi

# Ask if user wants to keep data
echo "Do you want to keep the data (tasks, settings)?"
echo "  1) Stop server but keep data (can restart later)"
echo "  2) Stop server and DELETE ALL DATA"
read -p "Enter choice [1]: " choice
choice=${choice:-1}

echo ""

if [ "$choice" = "2" ]; then
    echo "WARNING: This will delete all tasks and configuration!"
    read -p "Are you sure? (type 'yes' to confirm): " confirm
    if [ "$confirm" = "yes" ]; then
        echo "Stopping and removing all data..."
        $COMPOSE_CMD down -v
        echo "✓ Server stopped and all data deleted"
    else
        echo "Cancelled"
        exit 0
    fi
else
    echo "Stopping server (data will be preserved)..."
    $COMPOSE_CMD down
    echo "✓ Server stopped (data preserved)"
    echo ""
    echo "To restart: ./scripts/start-test-server.sh"
fi

echo ""
