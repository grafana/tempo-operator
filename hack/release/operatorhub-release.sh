#!/bin/bash
# Publish a released bundle to OperatorHub: opens a pull request against
# k8s-operatorhub/community-operators (community bundle) and against
# redhat-openshift-ecosystem/community-operators-prod (openshift bundle).
#
# This replaces the "Reusable - Create operator hub pull request" workflow and
# the "Publish to operatorHUB" (olm.yaml) workflow, which needed a bot personal
# access token stored as a repository secret. It runs with your own 'gh'
# credentials and your own fork of each OperatorHub repository instead.
#
# Run it after the Release workflow published the images to ghcr.io.

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

DRY_RUN=false
FORCE=false
VERSION=""

usage() {
    cat <<EOF
Usage: $(basename "$0") [version] [--dry-run] [--force]

Opens the OperatorHub pull requests for a released version of the operator.

Arguments:
  [version]    version to publish, without the leading "v" (e.g. 0.24.0).
               Defaults to the topmost version in CHANGELOG.md.

Options:
  --dry-run    stage the bundle and show the diff, but do not push or open PRs
  --force      continue even if the GitHub release or the v<version> tag is missing,
               and overwrite an already staged bundle directory
  -h, --help   show this help

Environment:
  TEMPO_RELEASE_CACHE_DIR
               where to keep the clones of the OperatorHub repositories
               (default: $CACHE_DIR)

The bundle is taken from the v<version> tag, not from your working tree. The
OperatorHub repositories are cloned into the cache directory and reused across
releases; a checkout of them sitting next to tempo-operator is used as a
reference to speed up that first clone, but is never modified.
EOF
}

while [[ $# -gt 0 ]]; do
    case "$1" in
        --dry-run) DRY_RUN=true ;;
        --force) FORCE=true ;;
        -h|--help) usage; exit 0 ;;
        -*) usage >&2; fail "unknown option: $1" ;;
        *)
            [[ -z "$VERSION" ]] || { usage >&2; fail "unexpected argument: $1"; }
            VERSION=$1
            ;;
    esac
    shift
done

require_commands git gh
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
if ! gh release view "v$VERSION" --repo "$UPSTREAM_REPO" >/dev/null 2>&1; then
    if [[ "$FORCE" == "true" ]]; then
        warn "release v$VERSION does not exist in $UPSTREAM_REPO, continuing anyway (--force)"
    else
        fail "release v$VERSION does not exist in $UPSTREAM_REPO yet, wait for the Release workflow to finish (or pass --force)"
    fi
fi

# Take the bundle from the tag rather than from the working tree, so the result
# does not depend on the state of the local checkout.
if upstream_remote=$(remote_for_repo "$UPSTREAM_REPO"); then
    git fetch --quiet --tags "$upstream_remote" || warn "could not fetch tags from $upstream_remote"
fi

WORKTREE=""
cleanup() {
    [[ -n "$WORKTREE" ]] || return 0
    git worktree remove --force "$WORKTREE" 2>/dev/null || true
}
trap cleanup EXIT

if git rev-parse --quiet --verify "refs/tags/v$VERSION" >/dev/null; then
    WORKTREE=$(mktemp -d -t tempo-operator-bundle-XXXXXX)
    rmdir "$WORKTREE"
    git worktree add --quiet --detach "$WORKTREE" "v$VERSION"
    BUNDLE_SRC=$WORKTREE
    info "using the bundle from tag v$VERSION"
elif [[ "$FORCE" == "true" ]]; then
    BUNDLE_SRC=$PWD
    warn "tag v$VERSION not found, using the bundle from the current working tree (--force)"
else
    fail "tag v$VERSION not found, fetch it first (or pass --force to use the working tree)"
fi

publish() {
    local target=$1
    local org_repo=${target%:*}
    local variant=${target#*:}
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
        # keep it around for the next release. If the repository happens to be
        # cloned next to tempo-operator, borrow its objects to avoid downloading
        # them again; --dissociate copies them so that checkout stays untouched
        # and can be removed later without breaking this clone.
        local reference=()
        local sibling
        if sibling=$(sibling_checkout "$org_repo"); then
            info "[$org_repo] found a local checkout at $sibling, using it as a reference"
            reference=(--reference-if-able "$sibling" --dissociate)
        fi

        info "[$org_repo] cloning into $clone_dir (this takes a while the first time)"
        mkdir -p "$CACHE_DIR"
        gh repo clone "$GH_USER/$repo" "$clone_dir" -- --filter=blob:none --quiet "${reference[@]}"
    fi

    git -C "$clone_dir" checkout --quiet -B "$BRANCH" "origin/$default_branch"
    git -C "$clone_dir" reset --quiet --hard "origin/$default_branch"
    git -C "$clone_dir" clean -qfd

    # Only the maintainers listed in ci.yaml can get a pull request merged
    # automatically by the OperatorHub pipeline.
    local ci_file="$clone_dir/operators/tempo-operator/ci.yaml"
    if [[ -f "$ci_file" ]] && ! grep -qE "^[[:space:]]*-[[:space:]]+${GH_USER}[[:space:]]*$" "$ci_file"; then
        warn "[$org_repo] $GH_USER is not listed in operators/tempo-operator/ci.yaml, the pull request will not be merged automatically"
    fi

    local dest="$clone_dir/operators/tempo-operator/$VERSION"
    if [[ -d "$dest" ]]; then
        if [[ "$FORCE" == "true" ]]; then
            warn "[$org_repo] $VERSION is already published, overwriting (--force)"
            rm -rf "$dest"
        else
            fail "[$org_repo] operators/tempo-operator/$VERSION already exists upstream, nothing to do"
        fi
    fi

    info "[$org_repo] copying the $variant bundle to operators/tempo-operator/$VERSION"
    mkdir -p "$dest"
    cp -R "$BUNDLE_SRC/bundle/$variant/." "$dest/"

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
    gh pr create \
        --repo "$org_repo" \
        --base "$default_branch" \
        --head "$GH_USER:$BRANCH" \
        --title "$MESSAGE" \
        --body "$BODY"
}

# Keep going if one of the two targets fails, so a failure on one hub does not
# block the other. errexit has to be disabled around the subshell rather than
# running it as an "if" condition: bash suppresses errexit inside a condition,
# and the subshell inherits that suppression even if it sets -e itself.
failed=()
for target in "${TARGETS[@]}"; do
    set +e
    ( set -e; publish "$target" )
    rc=$?
    set -e
    [[ $rc -eq 0 ]] || failed+=("${target%:*}")
done

if [[ ${#failed[@]} -gt 0 ]]; then
    fail "failed to publish to: ${failed[*]}"
fi

info "done"
