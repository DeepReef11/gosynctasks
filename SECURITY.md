# Security Guide

This document explains how to securely manage credentials in gosynctasks.

## Credential Storage Options

### 1. System Keyring (RECOMMENDED)

Store credentials in your OS keyring (macOS Keychain, Windows Credential Manager, Linux Secret Service/GNOME Keyring/KDE Wallet).

**Advantages:** Encrypted by OS, protected by system authentication, never in plain text, survives config backups/sharing.

```bash
gosynctasks credentials set nextcloud myuser --prompt
```

#### Multiple Backends with Different Credentials

**Config example:**
```yaml
backends:
  # Production Nextcloud
  nextcloud-prod:
    type: nextcloud
    enabled: true
    host: "nextcloud.example.com"
    username: "myuser"

  # Docker Test Nextcloud
  nextcloud-test:
    type: nextcloud
    enabled: true
    host: "localhost:8080"
    username: "admin"
    allow_http: true
    suppress_http_warning: true
```

**Store separate credentials:**
```bash
# Production
gosynctasks credentials set nextcloud-prod myuser --prompt

# Test
gosynctasks credentials set nextcloud-test admin --prompt
```

**Use different backends:**
```bash
# Use test (if it's your default_backend)
gosynctasks MyList

# Explicitly use production
gosynctasks --backend nextcloud-prod MyList

# Explicitly use test
gosynctasks --backend nextcloud-test MyList
```

Each backend name gets its own keyring entry. You can have as many backends as needed with completely separate credentials.


### 2. Environment Variables

Good for CI/CD and containers.

```bash
export GOSYNCTASKS_NEXTCLOUD_USERNAME=myuser
export GOSYNCTASKS_NEXTCLOUD_PASSWORD=secret
export GOSYNCTASKS_NEXTCLOUD_HOST=nextcloud.example.com
```

**Environment variable pattern:** `GOSYNCTASKS_{BACKEND_NAME}_{FIELD}`
- Backend name: Uppercase, hyphens → underscores
- Examples: `GOSYNCTASKS_NEXTCLOUD_PROD_USERNAME`, `GOSYNCTASKS_NEXTCLOUD_TEST_PASSWORD`

### 3. Config URL (LEGACY)

Plain text in config file - not recommended except for local testing:

```yaml
nextcloud:
  type: nextcloud
  enabled: true
  url: "nextcloud://username:password@nextcloud.example.com"
```

⚠️ **Never commit config files with plain text credentials to version control!**

## Credential Priority

When multiple sources exist: **Keyring > Environment Variables > Config URL**

This allows overriding config URL credentials without modifying the config file.

## Migration from Config URL to Keyring

```bash
# 1. Store in keyring
gosynctasks credentials set nextcloud myuser --prompt

# 2. Update config: Remove password from URL
# Before: url: "nextcloud://myuser:mypassword@nextcloud.example.com"
# After:  host: "nextcloud.example.com"
#         username: "myuser"

# 3. Verify
gosynctasks credentials get nextcloud myuser
gosynctasks nextcloud  # Test connection

# 4. Safe to commit (no credentials in config)
git add config.yaml
git commit -m "Migrate to keyring credentials"
```

## Platform-Specific Setup

**macOS:** Credentials in Keychain (view via Keychain Access.app, search "gosynctasks-")

**Windows:** Credentials in Credential Manager (Control Panel → Credential Manager → Windows Credentials)

**Linux:** Requires GNOME Keyring or KDE Wallet
```bash
# Install if needed
sudo apt-get install gnome-keyring  # Debian/Ubuntu
sudo dnf install gnome-keyring      # Fedora/RHEL
sudo pacman -S gnome-keyring         # Arch
```

If keyring unavailable, error message suggests environment variables as fallback.

## CI/CD Configuration

Use environment variables with CI/CD secrets:

**GitHub Actions:**
```yaml
env:
  GOSYNCTASKS_NEXTCLOUD_USERNAME: ${{ secrets.NEXTCLOUD_USER }}
  GOSYNCTASKS_NEXTCLOUD_PASSWORD: ${{ secrets.NEXTCLOUD_PASS }}
  GOSYNCTASKS_NEXTCLOUD_HOST: nextcloud.example.com
run: ./gosynctasks sync
```

**GitLab CI:**
```yaml
script:
  - export GOSYNCTASKS_NEXTCLOUD_USERNAME=$NEXTCLOUD_USER
  - export GOSYNCTASKS_NEXTCLOUD_PASSWORD=$NEXTCLOUD_PASS
  - ./gosynctasks sync
```

## Troubleshooting

**Keyring not available:**
- Linux: Install GNOME Keyring or KDE Wallet (see Platform-Specific Setup)
- Fallback: Use environment variables
- Headless servers: Environment variables recommended

**Permission denied:**
- Ensure user has keyring access
- Linux: Ensure D-Bus is running

**Credentials not found:**
```bash
# Check status
gosynctasks credentials get nextcloud

# Solutions (in order of preference):
gosynctasks credentials set nextcloud myuser --prompt  # Keyring
export GOSYNCTASKS_NEXTCLOUD_USERNAME=...              # Environment
# Or add URL to config (not recommended)
```

### Auto-sync will be disabled/Error: failed to get remote backend config

- Make sure the sync 

## Reporting Security Issues

Report vulnerabilities via GitHub Issues. Do not post details publicly without coordinating with maintainers first.
