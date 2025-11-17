#!/usr/bin/env python3
# Relative date formatter - shows "2 days ago", "in 3 days", etc.
# Usage: Reads task JSON from stdin, outputs formatted date to stdout

import sys
import json
from datetime import datetime, timezone

def format_relative_date(date_str):
    """Format a date as relative to now (e.g., '2 days ago', 'in 3 days')"""
    if not date_str:
        return ""

    try:
        # Parse ISO 8601 date
        date = datetime.fromisoformat(date_str.replace('Z', '+00:00'))
        now = datetime.now(timezone.utc)

        # Calculate difference
        delta = date - now
        days = delta.days

        if days == 0:
            return "Today"
        elif days == 1:
            return "Tomorrow"
        elif days == -1:
            return "Yesterday"
        elif days > 0:
            return f"in {days} days"
        else:
            return f"{abs(days)} days ago"
    except Exception as e:
        return date_str

def main():
    # Read JSON from stdin
    try:
        data = json.load(sys.stdin)

        # Get the date field (check for due_date, start_date, etc.)
        date_str = data.get('due_date') or data.get('start_date') or data.get('created')

        # Format and output
        result = format_relative_date(date_str)
        print(result)
    except Exception as e:
        print(f"Error: {e}", file=sys.stderr)
        sys.exit(1)

if __name__ == "__main__":
    main()
