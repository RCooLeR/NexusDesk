#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
APP_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
REPO_ROOT="$(cd "${APP_ROOT}/.." && pwd)"

export CGO_ENABLED=1
export GOFLAGS="-mod=readonly"
if [[ "$(uname -s)" == "Linux" ]]; then
  export LANG="C.UTF-8"
  export LC_ALL="C.UTF-8"
else
  export LANG="en_US.UTF-8"
  export LC_ALL="en_US.UTF-8"
fi

cd "${APP_ROOT}"

version="${NEXUSDESK_VERSION:-0.0.0-ci}"
commit="${GITHUB_SHA:-$(git -C "${REPO_ROOT}" rev-parse --short HEAD)}"
commit="${commit:0:12}"
build_date="${NEXUSDESK_BUILD_DATE:-$(date -u +%Y-%m-%dT%H:%M:%SZ)}"
ldflags="-X nexusdesk/internal/buildinfo.Version=${version} -X nexusdesk/internal/buildinfo.Commit=${commit} -X nexusdesk/internal/buildinfo.BuildDate=${build_date}"

cleanup() {
  rm -f "${APP_ROOT}/nexusdesk" "${APP_ROOT}/nexusdesk.exe"
  rm -f "${APP_ROOT}/build/nexusdesk" "${APP_ROOT}/build/nexusdesk.exe"
  rm -f "${APP_ROOT}/build/nexusdesk-"*-manifest.json
  rm -f "${APP_ROOT}/build/nexusdesk-"*-sbom.json
  rm -f "${APP_ROOT}/build/nexusdesk-"*-provenance.json
  rm -f "${APP_ROOT}/build/nexusdesk-"*.tar.gz
  rm -rf "${APP_ROOT}/build/nexusdesk-linux-"* "${APP_ROOT}/build/nexusdesk-macos-"*
}
trap cleanup EXIT

echo "Checking gofmt..."
go_files="$(git ls-files '*.go')"
if [[ -n "${go_files}" ]]; then
  unformatted="$(printf '%s\n' "${go_files}" | xargs gofmt -l)"
  if [[ -n "${unformatted}" ]]; then
    echo "${unformatted}" >&2
    echo "gofmt check failed." >&2
    exit 1
  fi
fi

echo "Running tests..."
go test ./...

echo "Running static analysis..."
go vet ./...

echo "Validating build metadata..."
go test -ldflags "${ldflags}" ./internal/buildinfo

echo "Building native executable..."
mkdir -p build
artifact_path="${APP_ROOT}/build/nexusdesk"
manifest_platform="$(go env GOOS)"
manifest_path="${APP_ROOT}/build/nexusdesk-${manifest_platform}-manifest.json"
sbom_path="${APP_ROOT}/build/nexusdesk-${manifest_platform}-sbom.json"
provenance_path="${APP_ROOT}/build/nexusdesk-${manifest_platform}-provenance.json"
go build -ldflags "${ldflags}" -o "${artifact_path}" .

echo "Generating release evidence..."
go run ./cmd/release-manifest -artifact "${artifact_path}" -output "${manifest_path}" -platform "${manifest_platform}" -version "${version}" -commit "${commit}" -build-date "${build_date}" -repository "RCooLeR/NexusDesk" -workflow "scripts/ci-unix.sh" -source-commit-full "${commit}"
test -s "${manifest_path}"
test -s "${sbom_path}"
test -s "${provenance_path}"

echo "Packaging native Unix artifact..."
bash scripts/package-unix.sh --output-dir build --version "${version}" --commit "${commit}" --build-date "${build_date}"

echo "Checking diff whitespace..."
git -C "${REPO_ROOT}" diff --check
