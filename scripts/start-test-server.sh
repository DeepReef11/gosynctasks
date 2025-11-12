#!/bin/bash
# Quick start script for Nextcloud test server

set -e

echo "=== Starting Nextcloud Test Server ==="
echo ""

# Check if docker is available
if ! command -v docker &> /dev/null; then
    echo "ERROR: Docker is not installed"
    echo "Please install Docker: https://docs.docker.com/get-docker/"
    exit 1
fi

# Check if docker compose is available
if docker compose version &> /dev/null; then
    COMPOSE_CMD="docker compose"
elif command -v docker-compose &> /dev/null; then
    COMPOSE_CMD="docker-compose"
else
    echo "ERROR: Docker Compose is not installed"
    echo "Please install Docker Compose: https://docs.docker.com/compose/install/"
    exit 1
fi

echo "Using: $COMPOSE_CMD"
echo ""

# Start the services
echo "Starting services..."
$COMPOSE_CMD up -d

echo ""
echo "Waiting for Nextcloud to be ready (this may take 30-60 seconds)..."

# Wait for Nextcloud to be healthy
TIMEOUT=120
ELAPSED=0
while [ $ELAPSED -lt $TIMEOUT ]; do
    if $COMPOSE_CMD exec -T nextcloud curl -sf http://localhost/status.php > /dev/null 2>&1; then
        echo ""
        echo "✓ Nextcloud is ready!"
        break
    fi
    echo -n "."
    sleep 2
    ELAPSED=$((ELAPSED + 2))
done

if [ $ELAPSED -ge $TIMEOUT ]; then
    echo ""
    echo "WARNING: Nextcloud is taking longer than expected to start"
    echo "Check logs with: $COMPOSE_CMD logs -f nextcloud"
    exit 1
fi

echo ""
echo "=== Nextcloud Test Server is Running ==="
echo ""
echo "Web Interface:  http://localhost:8080"
echo "Username:       admin"
echo "Password:       admin123"
echo ""
echo "CalDAV URL:     http://localhost:8080/remote.php/dav"
echo ""
echo "Add this to ~/.config/gosynctasks/config.json:"
echo ""
echo '  "connector": {'
echo '    "url": "nextcloud://admin:admin123@localhost:8080",'
echo '    "insecure_skip_verify": false'
echo '  }'
echo ""
echo "Commands:"
echo "  View logs:    $COMPOSE_CMD logs -f nextcloud"
echo "  Stop server:  $COMPOSE_CMD down"
echo "  Restart:      $COMPOSE_CMD restart"
echo ""
