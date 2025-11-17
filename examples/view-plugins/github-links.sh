#!/bin/sh
# Convert summary with GitHub issue references to clickable links
# Usage: Reads task JSON from stdin, outputs formatted summary with links

read -r input
summary=$(echo "$input" | jq -r '.summary')

# Replace #123 patterns with GitHub issue links
# Customize GITHUB_REPO to your repository
GITHUB_REPO="${GITHUB_REPO:-owner/repo}"

# Use sed to replace #NUMBER with links
echo "$summary" | sed -E "s/#([0-9]+)/https:\/\/github.com\/$GITHUB_REPO\/issues\/\1/g"
