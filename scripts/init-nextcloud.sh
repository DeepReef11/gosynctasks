#!/bin/bash
set -e

echo "=== Nextcloud Test Server Initialization ==="

# Wait for Nextcloud to be fully initialized
while [ ! -f /var/www/html/config/config.php ]; do
    echo "Waiting for Nextcloud to be initialized..."
    sleep 5
done

echo "Nextcloud is initialized. Installing Tasks app..."

# Install and enable the Tasks app
php /var/www/html/occ app:install tasks || echo "Tasks app already installed"
php /var/www/html/occ app:enable tasks

# Install and enable Calendar app (CalDAV dependency)
php /var/www/html/occ app:install calendar || echo "Calendar app already installed"
php /var/www/html/occ app:enable calendar

# Create a default task list for testing
echo "Creating default task list..."
# This will be created automatically when first accessed via CalDAV

echo "=== Initialization Complete ==="
echo ""
echo "Nextcloud Test Server is ready!"
echo "Access at: http://localhost:8080"
echo "Username: admin"
echo "Password: admin123"
echo ""
echo "CalDAV URL: http://localhost:8080/remote.php/dav"
echo ""
echo "Update your config.json with:"
echo '  "connector": {'
echo '    "url": "nextcloud://admin:admin123@localhost:8080",'
echo '    "insecure_skip_verify": false'
echo '  }'
echo ""
