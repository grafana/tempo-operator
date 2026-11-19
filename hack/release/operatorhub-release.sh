#!/bin/bash
# Publish a released bundle to OperatorHub: opens a pull request against
# k8s-operatorhub/community-operators (community bundle) and against
# redhat-openshift-ecosystem/community-operators-prod (openshift bundle).
#
# This replaces the "Publish to operatorHUB" and "Reusable - Create operator hub
# pull request" workflows, which needed a bot personal access token stored as a
# repository secret. It runs with your own 'gh' credentials and your own fork of
# each OperatorHub repository instead.
#
# Run it after the Release workflow published the images to ghcr.io. Re-running
# it is safe: a target that already has the version is skipped.

set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")/../.."
source hack/release/common.sh

# "<org>/<repo>:<bundle variant>"
TARGETS=(
    "k8s-operatorhub/community-operators:community"
    "redhat-openshift-ecosystem/community-operators-prod:openshift"
)
CACHE_DIR="${TEMPO_RELEASE_CACHE_DIR:-${XDG_CACHE_HOME:-$HOME/.cache}/tempo-operator-release}"
REVIEWERS="@pavolloffay @frzifus @andreasgerstmayr @rubenvp8510 @iblancasa"

USAGE="usage: $(basename "$0") [version] [--dry-run]
  [version]   version to publish, without the leading \"v\" (e.g. 0.24.0),
              defaults to the topmost version in CHANGELOG.md
  --dry-run   stage the bundle and show the diff, but do not push and do not
              open the pull requests
  \$TEMPO_RELEASE_CACHE_DIR
              where to keep the clones of the OperatorHub repositories
              (default: $CACHE_DIR)"

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

require_commands git gh tar
require_gh_auth

if [[ -z "$VERSION" ]]; then
    VERSION=$(version_from_changelog)
    [[ -n "$VERSION" ]] || fail "could not determine the version from CHANGELOG.md, pass it as an argument"
    info "no version given, using $VERSION from CHANGELOG.md"
fi
[[ "$VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]] || fail "version must be x.y.z without a leading 'v', got '$VERSION'"

GH_USER=$(gh api user -q .login)
BRANCH="update-tempo-operator-to-$VERSION"
MESSAGE="Update the tempo to $VERSION"
BODY="Release tempo-operator \`$VERSION\`.

cc $REVIEWERS
"

# The GitHub release is created by the Release workflow, which also pushes the
# bundle and catalog images. Its presence is what tells us the images the CSV
# refers to actually exist.
gh release view "v$VERSION" --repo "$UPSTREAM_REPO" >/dev/null 2>&1 \
    || fail "release v$VERSION does not exist in $UPSTREAM_REPO yet, wait for the Release workflow to finish"

# The bundle is taken from the tag rather than from the working tree, so the
# result does not depend on the state of the local checkout.
git fetch --quiet --tags || warn "could not fetch the tags"
git rev-parse --quiet --verify "refs/tags/v$VERSION" >/dev/null || fail "tag v$VERSION not found"

publish() {
    local org_repo=${1%:*} variant=${1#*:}
    local repo=${org_repo#*/}
    local clone_dir="$CACHE_DIR/$repo"

    info "[$org_repo] syncing $GH_USER/$repo"
    gh repo fork "$org_repo" --clone=false --remote=false >/dev/null
    gh repo sync "$GH_USER/$repo" --source "$org_repo" --force

    local default_branch
    default_branch=$(gh repo view "$org_repo" --json defaultBranchRef -q .defaultBranchRef.name)

    if [[ -d "$clone_dir/.git" ]]; then
        info "[$org_repo] updating the cached clone at $clone_dir"
        git -C "$clone_dir" fetch --quiet --prune origin
    else
        # A full clone of these repositories is large and a shallow one cannot be
        # pushed back ("shallow update not allowed"), so do a blobless clone and
        # keep it around for the next release.
        info "[$org_repo] cloning into $clone_dir (this takes a while the first time)"
        mkdir -p "$CACHE_DIR"
        gh repo clone "$GH_USER/$repo" "$clone_dir" -- --filter=blob:none --quiet
    fi
    git -C "$clone_dir" checkout --quiet --force -B "$BRANCH" "origin/$default_branch"
    git -C "$clone_dir" clean -qfd

    local dest="$clone_dir/operators/tempo-operator/$VERSION"
    if [[ -d "$dest" ]]; then
        info "[$org_repo] $VERSION is already published, skipping"
        return 0
    fi

    info "[$org_repo] copying the $variant bundle to operators/tempo-operator/$VERSION"
    mkdir -p "$dest"
    git archive "v$VERSION" "bundle/$variant" | tar -x --strip-components=2 -C "$dest"

    git -C "$clone_dir" add -A
    # The OperatorHub repositories enforce DCO, hence the sign-off.
    git -C "$clone_dir" commit --quiet -s -m "$MESSAGE"
    git -C "$clone_dir" --no-pager show --stat HEAD

    if [[ "$DRY_RUN" == "true" ]]; then
        info "[$org_repo] dry run: not pushing $BRANCH and not creating a pull request"
        return 0
    fi

    info "[$org_repo] pushing $BRANCH to $GH_USER/$repo"
    git -C "$clone_dir" push --quiet --force --set-upstream origin "$BRANCH"

    info "[$org_repo] creating the pull request"
    gh pr create --repo "$org_repo" --base "$default_branch" --head "$GH_USER:$BRANCH" \
        --title "$MESSAGE" --body "$BODY"
}

# A failure on one hub does not need to be isolated from the other: fix it and
# run the script again, the target that already succeeded is skipped.
for target in "${TARGETS[@]}"; do
    publish "$target"
done

info "done"
