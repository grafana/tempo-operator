#!/bin/bash
# Shared helpers for the release scripts in this directory.

UPSTREAM_REPO="grafana/tempo-operator"

info() { echo -e "\033[1;34m==>\033[0m $*"; }
warn() { echo -e "\033[1;33mwarning:\033[0m $*" >&2; }
fail() { echo -e "\033[1;31merror:\033[0m $*" >&2; exit 1; }

require_commands() {
    local cmd
    for cmd in "$@"; do
        command -v "$cmd" >/dev/null || fail "'$cmd' is required but not installed"
    done
}

require_gh_auth() {
    gh auth status >/dev/null 2>&1 || fail "not logged in to GitHub, run 'gh auth login'"
}

# Print the operator version of the topmost section of the CHANGELOG, which is
# how the release workflow determines the version being released.
version_from_changelog() {
    grep -Eo '## [0-9]+\.[0-9]+\.[0-9]+' CHANGELOG.md | head -n 1 | awk '{print $2}'
}
