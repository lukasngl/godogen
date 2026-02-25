#!/usr/bin/env sh

set -e

echo_ci() {
	test -z "$CI" || echo "$@"
}

echo_ci "::group::creating clean copy"
# Create a temporary directory and delete it once the script is done
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

# Create a copy of all *tracked* files in the temporary directory
git checkout-index --all --prefix "$TMP/"
cd "$TMP"

# Initializie as git repo to enable diff
git init . 2>/dev/null
git add . 2>/dev/null
echo_ci "::endgroup::"

# Run the given command, that may make changes to the source.
echo_ci "::group::running command"
"$@"
exit_code=$?
if [ $exit_code -ne 0 ]; then
	echo_ci "ERROR: Command '$*' failed with exit code $exit_code"
	echo_ci "::endgroup::"
	exit $exit_code
fi
echo_ci "::endgroup::"

# Print all changes and return with a non zero error if anything changed.
echo_ci "::group::check if anything changed"
git diff --exit-code --ignore-space-at-eol
exit_code=$?
if [ $exit_code -ne 0 ]; then
	echo_ci "ERROR: Command '$*' changed the source tree"
	echo_ci "Please run '$*' locally and commit the changes"
	echo_ci "::endgroup::"
	exit $exit_code
else
	echo_ci "No changes detected"
fi
echo_ci "::endgroup::"
