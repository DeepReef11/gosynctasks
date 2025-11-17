#!/bin/sh
# Format tags as colorful badges
# Usage: Reads task JSON from stdin, outputs formatted tags to stdout

read -r input
tags=$(echo "$input" | jq -r '.categories // [] | join(", ")')

if [ -z "$tags" ] || [ "$tags" = "null" ]; then
    echo ""
else
    # Convert comma-separated tags to badge format
    echo "$tags" | sed 's/\([^,]*\)/[\1]/g' | sed 's/, / /g'
fi
