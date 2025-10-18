#!/bin/bash

# Usage: ./scripts/bump-version.sh [major|minor|patch]

set -e

BUMP_TYPE=${1:-patch}
CURRENT_VERSION=$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")

# Remove 'v' prefix
CURRENT_VERSION=${CURRENT_VERSION#v}

# Split version
IFS='.' read -ra VERSION_PARTS <<< "$CURRENT_VERSION"
MAJOR=${VERSION_PARTS[0]}
MINOR=${VERSION_PARTS[1]}
PATCH=${VERSION_PARTS[2]}

# Bump version
case $BUMP_TYPE in
  major)
    MAJOR=$((MAJOR + 1))
    MINOR=0
    PATCH=0
    ;;
  minor)
    MINOR=$((MINOR + 1))
    PATCH=0
    ;;
  patch)
    PATCH=$((PATCH + 1))
    ;;
  *)
    echo "Usage: $0 [major|minor|patch]"
    exit 1
    ;;
esac

NEW_VERSION="v${MAJOR}.${MINOR}.${PATCH}"

echo "Bumping version from v${CURRENT_VERSION} to ${NEW_VERSION}"

# Update version.go
cat > version.go <<EOF
package snipgo

// Version information
const (
    Version = "${MAJOR}.${MINOR}.${PATCH}"
    Major   = ${MAJOR}
    Minor   = ${MINOR}
    Patch   = ${PATCH}
)
EOF

# Commit and tag
git add version.go
git commit -m "Bump version to ${NEW_VERSION}"
git tag -a ${NEW_VERSION} -m "Release ${NEW_VERSION}"

echo "Created tag ${NEW_VERSION}"
echo "Run 'git push origin main --tags' to publish"