#!/usr/bin/env bash
set -euo pipefail

VERSION="${1:-0.1.0}"
ARCH="${2:-amd64}"
MAINTAINER="${DEB_MAINTAINER:-K}"
HOMEPAGE="${HOMEPAGE:-https://github.com/krikera/rot13}"

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DIST_DIR="${ROOT_DIR}/dist"
PKG_DIR="${DIST_DIR}/pkg_deb"
BIN_NAME="rot13"
DEB_NAME="rot13_${VERSION}_${ARCH}.deb"

echo "Building rot13 binary for linux/${ARCH} (version: ${VERSION})..."
mkdir -p "${DIST_DIR}"
rm -rf "${PKG_DIR}"
mkdir -p "${PKG_DIR}/DEBIAN"
mkdir -p "${PKG_DIR}/usr/bin"
mkdir -p "${PKG_DIR}/usr/share/man/man1"
mkdir -p "${PKG_DIR}/usr/share/doc/rot13"

CGO_ENABLED=0 GOOS=linux GOARCH="${ARCH}" go build \
    -trimpath \
    -ldflags="-s -w -X main.version=${VERSION}" \
    -o "${PKG_DIR}/usr/bin/${BIN_NAME}" \
    "${ROOT_DIR}/cmd/rot13"

# Set base permissions
chmod 755 "${PKG_DIR}/usr/bin/${BIN_NAME}"

# Install man page from source
cp "${ROOT_DIR}/man/rot13.1" "${PKG_DIR}/usr/share/man/man1/rot13.1"
gzip -9 -n -f "${PKG_DIR}/usr/share/man/man1/rot13.1"

# Install shell completions
mkdir -p "${PKG_DIR}/usr/share/bash-completion/completions"
cp "${ROOT_DIR}/completions/rot13.bash" "${PKG_DIR}/usr/share/bash-completion/completions/rot13"
mkdir -p "${PKG_DIR}/usr/share/zsh/vendor-completions"
cp "${ROOT_DIR}/completions/rot13.zsh" "${PKG_DIR}/usr/share/zsh/vendor-completions/_rot13"
mkdir -p "${PKG_DIR}/usr/share/fish/vendor_completions.d"
cp "${ROOT_DIR}/completions/rot13.fish" "${PKG_DIR}/usr/share/fish/vendor_completions.d/rot13.fish"

# Debian copyright file (Machine-readable DEP-5 format)
cat << EOF > "${PKG_DIR}/usr/share/doc/rot13/copyright"
Format: https://www.debian.org/doc/packaging-manuals/copyright-format/1.0/
Upstream-Name: rot13
Source: ${HOMEPAGE}

Files: *
Copyright: 2026 rot13 contributors
License: MIT
 Permission is hereby granted, free of charge, to any person obtaining a copy
 of this software and associated documentation files (the "Software"), to deal
 in the Software without restriction, including without limitation the rights
 to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 copies of the Software, and to permit persons to whom the Software is
 furnished to do so, subject to the following conditions:
 .
 The above copyright notice and this permission notice shall be included in all
 copies or substantial portions of the Software.
 .
 THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 SOFTWARE.
EOF

# Debian changelog file
cat << EOF > "${PKG_DIR}/usr/share/doc/rot13/changelog.Debian"
rot13 (${VERSION}) unstable; urgency=medium

  * Initial release of rot13 command-line utility.

 -- ${MAINTAINER}  Fri, 09 Oct 2026 12:00:00 +0000
EOF

gzip -9 -n -f "${PKG_DIR}/usr/share/doc/rot13/changelog.Debian"

# Normalize directory and file permissions per Debian Policy Manual §10.9
find "${PKG_DIR}/usr" -type d -exec chmod 755 {} +
find "${PKG_DIR}/usr/share" -type f -exec chmod 644 {} +
chmod 755 "${PKG_DIR}/usr/bin/${BIN_NAME}"

# Calculate Installed-Size in KiB (Debian Policy Manual §5.6.20)
INSTALLED_SIZE=$(du -ks "${PKG_DIR}/usr" | cut -f1)

# Generate md5sums for all installed files (Debian Policy Manual §3.5)
(
    cd "${PKG_DIR}"
    find usr -type f -exec md5sum {} + > DEBIAN/md5sums
)
chmod 644 "${PKG_DIR}/DEBIAN/md5sums"

# Create Debian control file with Installed-Size
cat << EOF > "${PKG_DIR}/DEBIAN/control"
Package: rot13
Version: ${VERSION}
Section: utils
Priority: optional
Architecture: ${ARCH}
Standards-Version: 4.7.0
Installed-Size: ${INSTALLED_SIZE}
Maintainer: ${MAINTAINER}
Homepage: ${HOMEPAGE}
Description: ROT13 command-line utility
 A CLI tool to encode and decode files or standard input using
 the ROT13 symmetric substitution cipher.
EOF
chmod 644 "${PKG_DIR}/DEBIAN/control"

echo "Packaging ${DEB_NAME}..."
dpkg-deb --build --root-owner-group "${PKG_DIR}" "${DIST_DIR}/${DEB_NAME}"
rm -rf "${PKG_DIR}"

echo "Debian package created successfully: ${DIST_DIR}/${DEB_NAME}"
