#!/usr/bin/env bash
set -e

# Usage: ./scripts/release.sh <patch|minor|major> "Release summary"

INCREMENT=${1:-patch}
SUMMARY=${2:-"Update and improvements"}

echo "🔍 Running frontend typecheck..."
npm --prefix fe run typecheck

echo "🔍 Running backend tests..."
(cd be && go test ./...)

echo "📦 Reading current version from fe/package.json..."
CURRENT_VERSION=$(node -p "require('./fe/package.json').version")
IFS='.' read -r -a PARTS <<< "$CURRENT_VERSION"
MAJOR=${PARTS[0]}
MINOR=${PARTS[1]}
PATCH=${PARTS[2]}

if [ "$INCREMENT" = "major" ]; then
  MAJOR=$((MAJOR + 1))
  MINOR=0
  PATCH=0
elif [ "$INCREMENT" = "minor" ]; then
  MINOR=$((MINOR + 1))
  PATCH=0
else
  PATCH=$((PATCH + 1))
fi

NEW_VERSION="${MAJOR}.${MINOR}.${PATCH}"
echo "🚀 Bumping version: ${CURRENT_VERSION} -> ${NEW_VERSION}"

# Update fe/package.json
node -e "
const fs = require('fs');
const pkg = JSON.parse(fs.readFileSync('fe/package.json', 'utf8'));
pkg.version = '${NEW_VERSION}';
fs.writeFileSync('fe/package.json', JSON.stringify(pkg, null, 2) + '\n');
"

TODAY=$(date +%Y-%m-%d)
echo "📝 Please ensure CHANGELOG.md has an entry for [v${NEW_VERSION}] - ${TODAY}."

git add .
COMMIT_MSG="chore(release): v${NEW_VERSION} - ${SUMMARY}

- Version bump to ${NEW_VERSION}
- ${SUMMARY}"

git commit -m "$COMMIT_MSG"
git tag -a "v${NEW_VERSION}" -m "Release v${NEW_VERSION}: ${SUMMARY}"

echo "⬆️  Pushing to git origin with tags..."
git push origin $(git rev-parse --abbrev-ref HEAD) --tags

echo "✅ Release v${NEW_VERSION} successfully published to GitHub!"
