#!/bin/bash

set -euo pipefail

if [ -z "$1" ]; then
  echo "Usage: $0 <version>"
  exit 1
fi

VERSION=$1
REPO="ashebanow/rainbridge"
ARCHIVE_URL="https://github.com/${REPO}/archive/refs/tags/v${VERSION}.tar.gz"

# Download the archive and calculate SHA256
SHA256=$(curl -L "${ARCHIVE_URL}" | shasum -a 256 | awk '{print $1}')

cat <<EOF
class Rainbridge < Formula
  desc "A utility to import bookmarks into Karakeep from Raindrop.io"
  homepage "https://github.com/${REPO}"
  url "${ARCHIVE_URL}"
  sha256 "${SHA256}"
  license "MIT"

  depends_on "go"

  def install
    system "go", "build", "-ldflags", "-s -w", "-o", bin/"rainbridge", "./cmd/rainbridge"
  end

  test do
    # Basic test to ensure the binary runs
    system bin/"rainbridge", "--version"
  end
end
EOF
