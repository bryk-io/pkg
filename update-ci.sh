#!/usr/bin/env bash
#
# update-ci.sh - Update GitHub Actions to their latest versions
#
# This script scans all workflow files in .github/workflows/, extracts
# pinned actions (with SHA hashes), queries GitHub's API for the latest
# version tags, resolves them to full SHAs, and updates the workflow files.
#
# Usage:
#   ./update-ci.sh [options]
#
# Options:
#   -d, --dry-run    Show what would be updated without making changes
#   -v, --verbose    Show detailed output
#   -h, --help       Show this help message
#
# Requirements:
#   - curl
#   - python3
#   - gh (GitHub CLI) or a GitHub token in GITHUB_TOKEN env var

set -euo pipefail

WORKFLOW_DIR=".github/workflows"
GITHUB_API="https://api.github.com"
DRY_RUN=false
VERBOSE=false
CACHE_DIR=$(mktemp -d)

# Clean up cache on exit
trap 'rm -rf "$CACHE_DIR"' EXIT

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

usage() {
  cat <<EOF
Usage: $(basename "$0") [options]

Update all GitHub Actions in workflow files to their latest versions.

Options:
  -d, --dry-run    Show what would be updated without making changes
  -v, --verbose    Show detailed output
  -h, --help       Show this help message

Actions are matched by their uses: lines with SHA pins and version comments:
  uses: owner/action@sha # v1.2.3

The script fetches the latest version tag from GitHub and resolves it to
a full SHA hash, then updates both the SHA and version comment.
EOF
}

log_info() {
  echo -e "${BLUE}[INFO]${NC} $*"
}

log_ok() {
  echo -e "${GREEN}[OK]${NC} $*"
}

log_warn() {
  echo -e "${YELLOW}[WARN]${NC} $*"
}

log_error() {
  echo -e "${RED}[ERROR]${NC} $*"
}

log_verbose() {
  if [[ "$VERBOSE" == true ]]; then
    echo -e "  ${NC}$*${NC}"
  fi
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
  case $1 in
    -d|--dry-run)
      DRY_RUN=true
      shift
      ;;
    -v|--verbose)
      VERBOSE=true
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      log_error "Unknown option: $1"
      usage
      exit 1
      ;;
  esac
done

# Cache for version lookups (uses temp files)
cache_get() {
  local key="$1"
  local cache_file="${CACHE_DIR}/${key//\//_}"
  if [[ -f "$cache_file" ]]; then
    cat "$cache_file"
    return 0
  fi
  return 1
}

cache_set() {
  local key="$1"
  local value="$2"
  local cache_file="${CACHE_DIR}/${key//\//_}"
  echo "$value" > "$cache_file"
}

# Check for GitHub CLI or token
get_auth_header() {
  if command -v gh &>/dev/null && gh auth status &>/dev/null 2>&1; then
    echo "Authorization: Bearer $(gh auth token)"
  elif [[ -n "${GITHUB_TOKEN:-}" ]]; then
    echo "Authorization: Bearer $GITHUB_TOKEN"
  fi
}

# Get the latest version tag for a GitHub action (with caching)
get_latest_version() {
  local owner="$1"
  local repo="$2"
  local key="version_${owner}_${repo}"

  # Check cache
  local cached
  if cached=$(cache_get "$key"); then
    echo "$cached"
    return 0
  fi

  local auth
  auth=$(get_auth_header)

  # Fetch releases first (preferred for version tags)
  local url="${GITHUB_API}/repos/${owner}/${repo}/releases?per_page=10"
  local releases
  if [[ -n "$auth" ]]; then
    releases=$(curl -s -H "$auth" -H "Accept: application/vnd.github+json" "$url")
  else
    releases=$(curl -s -H "Accept: application/vnd.github+json" "$url")
  fi

  # Check for releases
  local tag
  tag=$(echo "$releases" | python3 -c "
import sys, json
try:
    data = json.load(sys.stdin)
    if isinstance(data, list) and len(data) > 0:
        for r in data:
            if not r.get('draft', True) and not r.get('prerelease', True):
                print(r['tag_name'])
                break
except:
    pass
" 2>/dev/null)

  if [[ -z "$tag" ]]; then
    # Fall back to tags
    url="${GITHUB_API}/repos/${owner}/${repo}/tags?per_page=20"
    local tags
    if [[ -n "$auth" ]]; then
      tags=$(curl -s -H "$auth" -H "Accept: application/vnd.github+json" "$url")
    else
      tags=$(curl -s -H "Accept: application/vnd.github+json" "$url")
    fi
    tag=$(echo "$tags" | python3 -c "
import sys, json, re
try:
    data = json.load(sys.stdin)
    if isinstance(data, list) and len(data) > 0:
        def semver_key(t):
            name = t['name'].lstrip('v')
            parts = re.split(r'[.\-]', name)
            result = []
            for p in parts:
                try:
                    result.append(int(p))
                except ValueError:
                    result.append(0)
            return result
        versioned = [t for t in data if re.match(r'^v?\d+\.\d+', t['name'])]
        if versioned:
            versioned.sort(key=semver_key, reverse=True)
            print(versioned[0]['name'])
        elif data:
            print(data[0]['name'])
except:
    pass
" 2>/dev/null)
  fi

  # Cache the result
  cache_set "$key" "$tag"
  echo "$tag"
}

# Resolve a tag to its full SHA (with caching)
resolve_tag_to_sha() {
  local owner="$1"
  local repo="$2"
  local tag="$3"
  local key="sha_${owner}_${repo}_${tag}"

  # Check cache
  local cached
  if cached=$(cache_get "$key"); then
    echo "$cached"
    return 0
  fi

  local auth
  auth=$(get_auth_header)

  local url="${GITHUB_API}/repos/${owner}/${repo}/git/ref/tags/${tag}"
  local response
  if [[ -n "$auth" ]]; then
    response=$(curl -s -H "$auth" -H "Accept: application/vnd.github+json" "$url")
  else
    response=$(curl -s -H "Accept: application/vnd.github+json" "$url")
  fi

  # Check if it's an annotated tag
  local object_type
  object_type=$(echo "$response" | python3 -c "
import sys, json
try:
    data = json.load(sys.stdin)
    print(data.get('object', {}).get('type', ''))
except:
    pass
" 2>/dev/null)

  local sha
  if [[ "$object_type" == "tag" ]]; then
    local tag_sha
    tag_sha=$(echo "$response" | python3 -c "
import sys, json
try:
    data = json.load(sys.stdin)
    print(data.get('object', {}).get('sha', ''))
except:
    pass
" 2>/dev/null)

      local tag_url="${GITHUB_API}/repos/${owner}/${repo}/git/tags/${tag_sha}"
      local tag_response
      if [[ -n "$auth" ]]; then
        tag_response=$(curl -s -H "$auth" -H "Accept: application/vnd.github+json" "$tag_url")
      else
        tag_response=$(curl -s -H "Accept: application/vnd.github+json" "$tag_url")
      fi

      sha=$(echo "$tag_response" | python3 -c "
import sys, json
try:
    data = json.load(sys.stdin)
    print(data.get('object', {}).get('sha', ''))
except:
    pass
" 2>/dev/null)
  else
    sha=$(echo "$response" | python3 -c "
import sys, json
try:
    data = json.load(sys.stdin)
    print(data.get('object', {}).get('sha', ''))
except:
    pass
" 2>/dev/null)
  fi

  # Cache the result
  cache_set "$key" "$sha"
  echo "$sha"
}

# Extract actions from workflow files
# Returns: file|line|owner/action[/subpath]|current_sha|current_version
extract_actions() {
  for workflow in "${WORKFLOW_DIR}"/*.yml "${WORKFLOW_DIR}"/*.yaml; do
    [[ -f "$workflow" ]] || continue

    local line_num=0
    while IFS= read -r line; do
      line_num=$((line_num + 1))

      # Match pinned actions with version comments
      if echo "$line" | grep -qE '^\s*uses:\s+[a-zA-Z0-9_.-]+/[a-zA-Z0-9_.-]+(/[a-zA-Z0-9_.-]+)*@[0-9a-f]{40}\s+#\s+v'; then
        local action_ref
        action_ref=$(echo "$line" | sed -E 's/.*uses: +([a-zA-Z0-9_.-]+\/[a-zA-Z0-9_.-]+(\/[a-zA-Z0-9_.-]+)*)@[0-9a-f]{40}.*/\1/')

        local current_sha
        current_sha=$(echo "$line" | sed -E 's/.*@([0-9a-f]{40}).*/\1/')

        local current_version
        current_version=$(echo "$line" | sed -E 's/.*# +(v[0-9][^ ]*).*/\1/')

        echo "${workflow}|${line_num}|${action_ref}|${current_sha}|${current_version}"
      fi
    done < "$workflow"
  done
}

# Main
main() {
  log_info "Scanning workflow files in ${WORKFLOW_DIR}/..."

  local actions
  actions=$(extract_actions)

  if [[ -z "$actions" ]]; then
    log_warn "No pinned actions found in workflow files."
    exit 0
  fi

  # Collect unique action paths (owner/repo) for deduplication
  local unique_actions
  unique_actions=$(echo "$actions" | while IFS='|' read -r _ _ action_ref _ _; do
    echo "$action_ref" | cut -d'/' -f1-2
  done | sort -u)

  local unique_count
  unique_count=$(echo "$unique_actions" | wc -l | tr -d ' ')
  local total_count
  total_count=$(echo "$actions" | wc -l | tr -d ' ')
  log_info "Found ${total_count} action references (${unique_count} unique actions)"

  # Fetch latest versions for all unique actions
  log_info "Fetching latest versions..."
  while IFS= read -r action_path; do
    local owner repo
    owner=$(echo "$action_path" | cut -d'/' -f1)
    repo=$(echo "$action_path" | cut -d'/' -f2)

    local latest
    latest=$(get_latest_version "$owner" "$repo")
    log_verbose "  ${action_path}: ${latest}"
  done < <(echo "$unique_actions")

  log_info "Processing updates..."
  local updated=0
  local unchanged=0
  local failed=0

  while IFS='|' read -r file line_num action_ref current_sha current_version; do
    local action_path
    action_path=$(echo "$action_ref" | cut -d'/' -f1-2)

    local owner repo
    owner=$(echo "$action_path" | cut -d'/' -f1)
    repo=$(echo "$action_path" | cut -d'/' -f2)

    local version_key="version_${owner}_${repo}"
    local latest_version
    latest_version=$(cache_get "$version_key" || echo "")

    if [[ -z "$latest_version" ]]; then
      log_warn "Could not find latest version for ${action_ref}"
      failed=$((failed + 1))
      continue
    fi

    if [[ "$latest_version" == "$current_version" ]]; then
      log_ok "${action_ref}: already at latest ${current_version}"
      unchanged=$((unchanged + 1))
      continue
    fi

    # Resolve SHA (cached)
    local new_sha
    new_sha=$(resolve_tag_to_sha "$owner" "$repo" "$latest_version")

    if [[ -z "$new_sha" ]]; then
      log_warn "Could not resolve ${latest_version} to SHA for ${action_ref}"
      failed=$((failed + 1))
      continue
    fi

    if [[ "$DRY_RUN" == true ]]; then
      log_info "[DRY RUN] ${action_ref}: ${current_version} -> ${latest_version}"
      log_verbose "  ${current_sha} -> ${new_sha}"
    else
      if sed -i '' "${line_num}s|${action_ref}@${current_sha} # ${current_version}|${action_ref}@${new_sha} # ${latest_version}|" "$file"; then
        log_ok "${action_ref}: ${current_version} -> ${latest_version}"
      else
        log_error "Failed to update ${action_ref} in ${file}"
        failed=$((failed + 1))
        continue
      fi
    fi

    updated=$((updated + 1))
  done <<< "$actions"

  echo ""
  log_info "Summary: ${updated} updated, ${unchanged} unchanged, ${failed} failed"

  if [[ "$DRY_RUN" == true ]]; then
    log_info "This was a dry run. Run without -d/--dry-run to apply changes."
  fi
}

main
