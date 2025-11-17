#!/bin/sh
# Custom status formatter with emoji icons
# Usage: Reads task JSON from stdin, outputs formatted status to stdout

read -r input
status=$(echo "$input" | jq -r '.status')

case "$status" in
    "TODO"|"NEEDS-ACTION")
        echo "⏳ TODO"
        ;;
    "DONE"|"COMPLETED")
        echo "✅ DONE"
        ;;
    "PROCESSING"|"IN-PROCESS")
        echo "🔄 IN PROGRESS"
        ;;
    "CANCELLED")
        echo "❌ CANCELLED"
        ;;
    *)
        echo "📝 $status"
        ;;
esac
