#!/usr/bin/env bash
set -euo pipefail

VERSION="${VERSION:-1.0.0}"
BUILD_DATE="$(date -u '+%Y-%m-%dT%H:%M:%SZ')"
COMMIT="$(git rev-parse HEAD)"
DIST_DIR="dist"

echo "Building ADP ${VERSION}"
echo "Commit: ${COMMIT}"
echo "Build date: ${BUILD_DATE}"
echo ""

# Clean previous builds
rm -rf "${DIST_DIR}"
mkdir -p "${DIST_DIR}"

# Build matrix
PLATFORMS=(
    "linux/amd64"
    "linux/arm64"
    "darwin/amd64"
    "darwin/arm64"
    "windows/amd64"
)

for platform in "${PLATFORMS[@]}"; do
    IFS='/' read -r os arch <<< "${platform}"

    output_name="adp-${VERSION}-${os}-${arch}"
    if [ "${os}" = "windows" ]; then
        output_name="${output_name}.exe"
    fi

    echo "Building ${os}/${arch}..."
    GOOS="${os}" GOARCH="${arch}" go build \
        -ldflags "-X github.com/karoc/adp/internal/cli.Version=${VERSION} \
                  -X github.com/karoc/adp/internal/cli.Commit=${COMMIT} \
                  -X github.com/karoc/adp/internal/cli.BuildDate=${BUILD_DATE}" \
        -o "${DIST_DIR}/${output_name}" \
        ./cmd/adp

    echo "  ✓ ${output_name}"
done

echo ""
echo "Generating checksums..."
cd "${DIST_DIR}"
sha256sum adp-* > SHA256SUMS
cd ..

echo "  ✓ SHA256SUMS"
echo ""
echo "Build complete. Artifacts in ${DIST_DIR}/"
ls -lh "${DIST_DIR}"
