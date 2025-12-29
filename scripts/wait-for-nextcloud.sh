#!/bin/bash
# Wait for Nextcloud to be fully ready
# Used by CI and local testing

set -e

MAX_RETRIES=${MAX_RETRIES:-40}
RETRY_INTERVAL=${RETRY_INTERVAL:-2}
NEXTCLOUD_URL=${NEXTCLOUD_URL:-http://localhost:8080}

echo "Waiting for Nextcloud to be ready at $NEXTCLOUD_URL..."

# First wait for initial container startup
echo "Initial startup delay..."
sleep 10

# Wait for status.php endpoint
RETRIES=0
until curl -f -s "$NEXTCLOUD_URL/status.php" > /dev/null 2>&1; do
    RETRIES=$((RETRIES + 1))
    if [ $RETRIES -ge $MAX_RETRIES ]; then
        echo ""
        echo "ERROR: Nextcloud did not become ready after $MAX_RETRIES attempts"
        echo ""
        echo "=== Docker container status ==="
        docker ps -a
        echo ""
        echo "=== Nextcloud logs (last 50 lines) ==="
        docker-compose logs --tail=50 nextcloud
        exit 1
    fi
    echo -n "."
    sleep $RETRY_INTERVAL
done

echo ""
echo "✓ Nextcloud is responding"

# Verify CalDAV endpoint
echo "Verifying CalDAV endpoint..."
if curl -u admin:admin123 -X PROPFIND "$NEXTCLOUD_URL/remote.php/dav/calendars/admin/" \
     -H "Depth: 1" -s -o /dev/null -w "%{http_code}" | grep -q "207"; then
    echo "✓ CalDAV endpoint is accessible"
else
    echo "Warning: CalDAV endpoint returned unexpected status"
    echo "Continuing anyway..."
fi

# Additional health checks
echo "Running health checks..."

# Check if Nextcloud is installed
if curl -f -s "$NEXTCLOUD_URL/status.php" | grep -q '"installed":true'; then
    echo "✓ Nextcloud is installed"
else
    echo "Warning: Nextcloud may not be fully installed"
fi

# Check if we can authenticate
if curl -u admin:admin123 -f -s "$NEXTCLOUD_URL/ocs/v1.php/cloud/users/admin" -H "OCS-APIRequest: true" > /dev/null 2>&1; then
    echo "✓ Admin authentication successful"
else
    echo "Warning: Admin authentication failed"
fi

echo ""
echo "✓ Nextcloud is ready for testing!"
echo ""
