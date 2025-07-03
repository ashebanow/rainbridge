#!/bin/bash

set -euo pipefail

if [ -z "$1" ]; then
  echo "Usage: $0 <version>"
  exit 1
fi

VERSION=${VERSION}
TAP_REPO="ashebanow/homebrew-tools"
FORMULA_NAME="rainbridge.rb"

# Check if gh CLI is installed
if ! command -v gh &> /dev/null
then
    echo "Error: GitHub CLI (gh) is not installed. Please install it from cli.github.com"
    exit 1
fi

TEMP_DIR=$(mktemp -d)

echo "Cloning ${TAP_REPO} into ${TEMP_DIR}..."
git clone "git@github.com:${TAP_REPO}.git" "${TEMP_DIR}"

echo "Copying ${FORMULA_NAME} to ${TEMP_DIR}..."
cp "${FORMULA_NAME}" "${TEMP_DIR}/${FORMULA_NAME}"

cd "${TEMP_DIR}"

BRANCH_NAME="update-${FORMULA_NAME}-v${VERSION}"

echo "Creating new branch ${BRANCH_NAME}..."
git checkout -b "${BRANCH_NAME}"

echo "Adding and committing changes..."
git add "${FORMULA_NAME}"
git commit -m "feat(${FORMULA_NAME}): Update to v${VERSION}"

echo "Pushing branch to origin..."
git push -u origin "${BRANCH_NAME}"

echo "Creating pull request..."
gh pr create --repo "${TAP_REPO}" --title "feat(${FORMULA_NAME}): Update to v${VERSION}" --body "Updates ${FORMULA_NAME} to version v${VERSION}."

cd -

echo "Cleaning up temporary directory ${TEMP_DIR}..."
rm -rf "${TEMP_DIR}"

echo "Pull request process initiated. Check your GitHub repository: https://github.com/${TAP_REPO}/pulls"
