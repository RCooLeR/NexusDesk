#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
APP_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
REPO_ROOT="$(cd "${APP_ROOT}/.." && pwd)"

export CGO_ENABLED=1
export GOFLAGS="-mod=readonly"

cd "${APP_ROOT}"

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

echo "Building native executable..."
mkdir -p build
go build -o build/nexusdesk .

echo "Checking diff whitespace..."
git -C "${REPO_ROOT}" diff --check
