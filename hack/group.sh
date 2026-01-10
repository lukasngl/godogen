#!/usr/bin/env sh

# Run a command and wrap it with a github-actions group

set -e

if [ $# -lt 3 ]; then
	echo "Usage: $0 <icon> <group_name> <command> [args...]" >&2
	echo "  icon can be an emoji or empty string" >&2
	exit 1
fi

icon="$1"
group_name="$2"
shift 2

# Record start time
start_time=$(date +%s)

# Create temp file for logs
log_file=$(mktemp)
trap 'rm -f "$log_file"' EXIT

# Print start group
if [ -n "$CI" ]; then
	printf "::group::%s\n" "$group_name"
	# Execute command directly in CI mode
	"$@"
	exit_code=$?
	if [ $exit_code -ne 0 ]; then
		printf "::endgroup::\n"
	fi
	exit $exit_code
elif [ -n "$VERBOSE" ]; then
	# Verbose mode - show output but with nice formatting
	if [ -n "$icon" ]; then
		printf "\033[1;34m%s %s\033[0m\n" "$icon" "$group_name"
	else
		printf "\033[1;34m%s\033[0m\n" "$group_name"
	fi
	"$@"
	exit_code=$?
	end_time=$(date +%s)
	duration=$((end_time - start_time))
	if [ $exit_code -eq 0 ]; then
		printf "\033[32m✓\033[0m \033[2m%s (%ds)\033[0m\n" "$group_name" "$duration"
	else
		printf "❌ \033[2m%s failed (%ds)\033[0m\n" "$group_name" "$duration"
	fi
	echo # blank line
	exit $exit_code
else
	# Interactive mode - show single line
	if [ -n "$icon" ]; then
		printf "%s %s" "$icon" "$group_name"
	else
		printf "%s" "$group_name"
	fi

	# Execute command with output redirected
	if "$@" >"$log_file" 2>&1; then
		# Success - update line with checkmark and timing
		end_time=$(date +%s)
		duration=$((end_time - start_time))
		printf "\r\033[32m✓\033[0m \033[2m%s (%ds)\033[0m\n" "$group_name" "$duration"
	else
		# Failure - show full output
		exit_code=$?
		end_time=$(date +%s)
		duration=$((end_time - start_time))
		printf "\r❌ \033[2m%s failed (%ds)\033[0m\n" "$group_name" "$duration"
		echo # blank line
		cat "$log_file"
		echo # blank line
		exit 1
	fi
fi
