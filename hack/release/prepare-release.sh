#!/bin/bash
# Prepare a new release: bump OPERATOR_VERSION, regenerate the bundles and the
# CHANGELOG, and open the "Prepare release vx.y.z" pull request from your fork.
#
# This replaces the "Prepare the release" GitHub workflow, which needed a bot
# personal access token stored as a repository secret. It runs with your own
# 'gh' credentials instead.

set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")/../.."
source hack/release/common.sh

DRY_RUN=false
VERSION=""

usage() {
    cat <<EOF
Usage: $(basename "$0") <version> [--dry-run]

Prepares the release pull request for the Tempo Operator.

Arguments:
  <version>    version to release, without the leading "v" (e.g. 0.24.0)

Options:
  --dry-run    generate and commit everything locally, but do not push or open a PR
  -h, --help   show this help

Environment:
  FORK_REMOTE  name of the git remote pointing at your fork of $UPSTREAM_REPO
               (autodetected when unset)
EOF
}

while [[ $# -gt 0 ]]; do
    case "$1" in
        --dry-run) DRY_RUN=true ;;
        -h|--help) usage; exit 0 ;;
        -*) usage >&2; fail "unknown option: $1" ;;
        *)
            [[ -z "$VERSION" ]] || { usage >&2; fail "unexpected argument: $1"; }
            VERSION=$1
            ;;
    esac
    shift
done

[[ -n "$VERSION" ]] || { usage >&2; fail "no version given"; }
[[ "$VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || fail "version must be x.y.z without a leading 'v', got '$VERSION'"

require_commands git gh go make
require_gh_auth

BRANCH="releases/$VERSION"
UPSTREAM_REMOTE=$(remote_for_repo "$UPSTREAM_REPO") \
    || fail "no git remote points at $UPSTREAM_REPO"

if [[ -n "${FORK_REMOTE:-}" ]]; then
    git remote get-url "$FORK_REMOTE" >/dev/null 2>&1 || fail "FORK_REMOTE '$FORK_REMOTE' is not a git remote"
else
    for remote in $(git remote); do
        if [[ "$remote" != "$UPSTREAM_REMOTE" ]]; then
            FORK_REMOTE=$remote
            break
        fi
    done
    [[ -n "${FORK_REMOTE:-}" ]] \
        || fail "could not find a remote for your fork, set FORK_REMOTE=<remote> (or add one with 'gh repo fork $UPSTREAM_REPO')"
fi
FORK_REPO=$(repo_slug "$(git remote get-url "$FORK_REMOTE")")

info "releasing v$VERSION, pushing to $FORK_REPO (remote '$FORK_REMOTE')"

# Sanity checks: the release must be cut from an up-to-date, clean main.
[[ "$(git branch --show-current)" == "main" ]] || fail "not on the main branch"
[[ -z "$(git status --porcelain)" ]] || fail "working tree is not clean"
if git show-ref --quiet "refs/heads/$BRANCH"; then
    fail "branch $BRANCH already exists locally"
fi

info "fetching $UPSTREAM_REMOTE"
git fetch --quiet "$UPSTREAM_REMOTE" main
[[ "$(git rev-parse HEAD)" == "$(git rev-parse "$UPSTREAM_REMOTE/main")" ]] \
    || fail "local main differs from $UPSTREAM_REMOTE/main, pull the latest changes first"

info "bumping OPERATOR_VERSION to $VERSION in the Makefile"
sed -i "s/^OPERATOR_VERSION ?= .*/OPERATOR_VERSION ?= $VERSION/" Makefile
grep -q "^OPERATOR_VERSION ?= $VERSION$" Makefile || fail "failed to update OPERATOR_VERSION in the Makefile"

# IMG_PREFIX is passed explicitly: it is commonly exported to a personal
# registry for the e2e tests, and it ends up in the containerImage of the CSV.
info "generating the community and openshift bundles"
make bundle OPERATOR_VERSION="$VERSION" IMG_PREFIX="ghcr.io/$UPSTREAM_REPO"

info "generating the CHANGELOG"
make chlog-update OPERATOR_VERSION="$VERSION"

info "committing to $BRANCH"
git checkout --quiet -b "$BRANCH"
git add -A
git commit --quiet -m "Prepare Release $VERSION"
git --no-pager show --stat HEAD

if [[ "$DRY_RUN" == "true" ]]; then
    info "dry run: not pushing $BRANCH and not creating a pull request"
    info "to undo: git checkout main && git branch -D $BRANCH"
    exit 0
fi

info "pushing $BRANCH to $FORK_REMOTE"
git push --quiet --force --set-upstream "$FORK_REMOTE" "$BRANCH"

# The changelog CI job requires every pull request to add a .chloggen entry.
# This one consumes them instead, so the title is prefixed with [chore] to skip
# that check.
info "creating the pull request against $UPSTREAM_REPO"
head="$BRANCH"
[[ "$FORK_REPO" == "$UPSTREAM_REPO" ]] || head="${FORK_REPO%%/*}:$BRANCH"
gh pr create \
    --repo "$UPSTREAM_REPO" \
    --base main \
    --head "$head" \
    --title "[chore] Prepare release v$VERSION" \
    --body "v$VERSION"

cat <<EOF

Next steps:
  1. Run the OpenShift e2e tests against the $BRANCH branch (see RELEASE.md).
  2. Merge the pull request. The Release workflow creates the tag, the GitHub
     release and pushes the images to ghcr.io.
  3. Run ./hack/release/operatorhub-release.sh to open the OperatorHub pull requests.
EOF
