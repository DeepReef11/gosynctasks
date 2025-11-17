#!/bin/sh
# Custom priority formatter with visual indicators
# Usage: Reads task JSON from stdin, outputs formatted priority to stdout

read -r input
priority=$(echo "$input" | jq -r '.priority')

case "$priority" in
    1)
        echo "🔥🔥🔥 P1 CRITICAL"
        ;;
    2)
        echo "🔥🔥 P2 HIGH"
        ;;
    3)
        echo "🔥 P3 HIGH"
        ;;
    4|5|6)
        echo "📌 P$priority MEDIUM"
        ;;
    7|8|9)
        echo "💤 P$priority LOW"
        ;;
    0)
        echo "⚪ P0 NONE"
        ;;
    *)
        echo "P$priority"
        ;;
esac
