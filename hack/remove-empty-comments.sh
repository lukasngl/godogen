#!/usr/bin/env sh

# Remove empty Go comments (blank lines followed by //)
# This is a common pattern that can be cleaned up

set -e

# Check if ripgrep is available
if ! command -v rg >/dev/null 2>&1; then
	exit 0
fi

# Find files with empty comment pattern and remove them
rg -U '^$\n^//$' -l 2>/dev/null | while read -r file; do
	# Use sed to remove: blank line followed by // line
	sed -i '/^$/{N;/^\n\/\/$/d;}' "$file"
done
