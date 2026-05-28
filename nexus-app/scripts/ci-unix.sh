#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
APP_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
REPO_ROOT="$(cd "${APP_ROOT}/.." && pwd)"

export CGO_ENABLED=1
export GOFLAGS="-mod=readonly"

cd "${APP_ROOT}"

version="${NEXUSDESK_VERSION:-0.0.0-ci}"
commit="${GITHUB_SHA:-$(git -C "${REPO_ROOT}" rev-parse --short HEAD)}"
commit="${commit:0:12}"
build_date="${NEXUSDESK_BUILD_DATE:-$(date -u +%Y-%m-%dT%H:%M:%SZ)}"
ldflags="-X nexusdesk/internal/buildinfo.Version=${version} -X nexusdesk/internal/buildinfo.Commit=${commit} -X nexusdesk/internal/buildinfo.BuildDate=${build_date}"

cleanup() {
  rm -f "${APP_ROOT}/nexusdesk" "${APP_ROOT}/nexusdesk.exe"
  rm -f "${APP_ROOT}/build/nexusdesk" "${APP_ROOT}/build/nexusdesk.exe"
}
trap cleanup EXIT

echo "Checking gofmt..."
mapfile -t go_files < <(git ls-files '*.go')
if ((${#go_files[@]} > 0)); then
  unformatted="$(gofmt -l "${go_files[@]}")"
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
go build -ldflags "${ldflags}" -o build/nexusdesk .

echo "Checking diff whitespace..."
git -C "${REPO_ROOT}" diff --check
