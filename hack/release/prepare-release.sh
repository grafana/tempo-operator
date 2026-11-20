#!/bin/bash
# Prepare a new release: bump OPERATOR_VERSION, regenerate the bundles and the
# CHANGELOG, and open the "[chore] Prepare release vx.y.z" pull request.
#
# This replaces the "Prepare the release" GitHub workflow, which needed a bot
# personal access token stored as a repository secret. It runs with your own
# 'gh' credentials instead.

set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")/../.."
source hack/release/common.sh

USAGE="usage: $(basename "$0") <version> [--dry-run]
  <version>   version to release, without the leading \"v\" (e.g. 0.24.0)
  --dry-run   commit locally, but do not push and do not open a pull request"

DRY_RUN=false
VERSION=""
while [[ $# -gt 0 ]]; do
    case "$1" in
        --dry-run) DRY_RUN=true ;;
        -h|--help) echo "$USAGE"; exit 0 ;;
        -*) fail "unknown option: $1
$USAGE" ;;
        *) [[ -z "$VERSION" ]] || fail "unexpected argument: $1"; VERSION=$1 ;;
    esac
    shift
done
[[ "$VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || fail "version must be x.y.z without a leading 'v'
$USAGE"

require_commands git gh go make
require_gh_auth

# "[chore]" makes the changelog workflow skip this pull request, which consumes
# the .chloggen entries instead of adding one. The release workflow in turn only
# cuts a release for a commit whose message contains "[chore] Prepare release v",
# so that no other change to the CHANGELOG reaching main can trigger one. Commit
# message and pull request title are identical, so that holds after squashing.
TITLE="[chore] Prepare release v$VERSION"
BRANCH="releases/$VERSION"
REMOTE=$(git remote -v | awk -v r="$UPSTREAM_REPO" '$2 ~ r && $3 == "(fetch)" { print $1; exit }')
[[ -n "$REMOTE" ]] || fail "no git remote points at $UPSTREAM_REPO"

[[ "$(git branch --show-current)" == "main" ]] || fail "not on the main branch"
[[ -z "$(git status --porcelain)" ]] || fail "working tree is not clean"
! git show-ref --quiet "refs/heads/$BRANCH" || fail "branch $BRANCH already exists locally"
# Every branch of the repository requires signed commits; check before the slow part.
[[ "$(git config --get commit.gpgsign)" == "true" || -n "$(git config --get user.signingkey)" ]] \
    || fail "the repository requires signed commits, configure user.signingkey"

info "releasing v$VERSION to $UPSTREAM_REPO (remote '$REMOTE')"
git fetch --quiet "$REMOTE" main
[[ "$(git rev-parse HEAD)" == "$(git rev-parse "$REMOTE/main")" ]] \
    || fail "local main differs from $REMOTE/main, pull the latest changes first"

info "bumping OPERATOR_VERSION to $VERSION in the Makefile"
sed -i "s/^OPERATOR_VERSION ?= .*/OPERATOR_VERSION ?= $VERSION/" Makefile
grep -q "^OPERATOR_VERSION ?= $VERSION$" Makefile || fail "failed to update OPERATOR_VERSION in the Makefile"

# IMG_PREFIX is passed explicitly: it is commonly exported to a personal
# registry for the e2e tests, and it ends up in the containerImage of the CSV.
info "generating the community and openshift bundles, and the CHANGELOG"
make bundle OPERATOR_VERSION="$VERSION" IMG_PREFIX="ghcr.io/$UPSTREAM_REPO"
make chlog-update OPERATOR_VERSION="$VERSION"

info "committing to $BRANCH"
git checkout --quiet -b "$BRANCH"
git add -A
git commit --quiet --gpg-sign -m "$TITLE"
git --no-pager show --stat HEAD

if [[ "$DRY_RUN" == "true" ]]; then
    info "dry run: not pushing. To undo: git checkout main && git branch -D $BRANCH"
    exit 0
fi

info "pushing $BRANCH and creating the pull request"
git push --quiet --force --set-upstream "$REMOTE" "$BRANCH"
gh pr create --repo "$UPSTREAM_REPO" --base main --head "$BRANCH" --title "$TITLE" --body "v$VERSION"

cat <<EOF

Next steps:
  1. Run the OpenShift e2e tests against the $BRANCH branch (see RELEASE.md).
  2. Merge the pull request. The Release workflow creates the tag, the GitHub
     release and pushes the images to ghcr.io.
  3. Run ./hack/release/operatorhub-release.sh to open the OperatorHub pull requests.
EOF
