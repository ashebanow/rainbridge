class Rainbridge < Formula
  desc "A utility to import bookmarks into Karakeep from Raindrop.io"
  homepage "https://github.com/ashebanow/rainbridge"
  url "https://github.com/ashebanow/rainbridge/archive/refs/tags/vVERSION=0.1.0.tar.gz"
  sha256 "d5558cd419c8d46bdc958064cb97f963d1ea793866414c025906ec15033512ed"
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
