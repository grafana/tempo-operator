#!/bin/bash
# Shared helpers for the release scripts in this directory.

UPSTREAM_REPO="grafana/tempo-operator"

info() { echo -e "\033[1;34m==>\033[0m $*"; }
warn() { echo -e "\033[1;33mwarning:\033[0m $*" >&2; }
fail() { echo -e "\033[1;31merror:\033[0m $*" >&2; exit 1; }

# Print "owner/repo" for a git remote URL, both for SSH and HTTPS remotes.
repo_slug() {
    local url=${1%.git}
    url=${url%/}
    local repo=${url##*/}
    local owner=${url%/*}
    owner=${owner##*/}  # https://github.com/owner -> owner
    owner=${owner##*:}  # git@github.com:owner     -> owner
    echo "$owner/$repo"
}

# Print the name of the git remote pointing at the given "owner/repo",
# or return 1 if there is none.
remote_for_repo() {
    local wanted=$1 remote
    for remote in $(git remote); do
        if [[ "$(repo_slug "$(git remote get-url "$remote")")" == "$wanted" ]]; then
            echo "$remote"
            return 0
        fi
    done
    return 1
}

# Print the path of a local checkout of "owner/repo" sitting next to this
# repository, or return 1 if there is none. Developers often have the
# OperatorHub repositories cloned alongside tempo-operator; such a checkout can
# be used as a reference to speed up the first clone.
sibling_checkout() {
    local wanted=$1
    local dir="$(cd .. && pwd)/${wanted#*/}"
    [[ -d "$dir/.git" ]] || return 1

    local remote
    for remote in $(git -C "$dir" remote 2>/dev/null); do
        if [[ "$(repo_slug "$(git -C "$dir" remote get-url "$remote")")" == */"${wanted#*/}" ]]; then
            echo "$dir"
            return 0
        fi
    done
    return 1
}

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
# how the release workflows determine the version being released.
version_from_changelog() {
    grep -Eo '## [0-9]+\.[0-9]+\.[0-9]+' CHANGELOG.md | head -n 1 | awk '{print $2}'
}
