#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
APP_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

output_dir="${APP_ROOT}/build"
version="${NEXUSDESK_VERSION:-0.0.0-ci}"

while [[ $# -gt 0 ]]; do
  case "$1" in
    -o|--output-dir)
      output_dir="$2"
      shift 2
      ;;
    -v|--version)
      version="$2"
      shift 2
      ;;
    *)
      echo "Unknown argument: $1" >&2
      exit 2
      ;;
  esac
done

goos="$(go env GOOS)"
goarch="$(go env GOARCH)"
case "${goos}" in
  darwin)
    platform="macos"
    package_ext="zip"
    ;;
  linux)
    platform="linux"
    package_ext="tar.gz"
    ;;
  *)
    echo "smoke-unix-package.sh supports linux and macOS, got ${goos}." >&2
    exit 2
    ;;
esac

safe_version="$(printf '%s' "${version}" | sed 's/[^0-9A-Za-z._+-]/-/g')"
package_base="nexusdesk-${platform}-${goarch}-${safe_version}"
output_root="$(cd "${output_dir}" && pwd -P)"
package_path="${output_root}/${package_base}.${package_ext}"
smoke_root="${output_root}/package-smoke-${package_base}"
extract_dir="${smoke_root}/extract"
workspace_dir="${smoke_root}/workspace"

if [[ ! -s "${package_path}" ]]; then
  echo "Package artifact was not found: ${package_path}" >&2
  exit 1
fi

case "${smoke_root}" in
  "${output_root}"/*) ;;
  *)
    echo "Refusing to remove smoke path outside output directory: ${smoke_root}" >&2
    exit 2
    ;;
esac
rm -rf "${smoke_root}"
mkdir -p "${extract_dir}" "${workspace_dir}"

if [[ "${platform}" == "macos" ]]; then
  ditto -x -k "${package_path}" "${extract_dir}"
  executable="${extract_dir}/${package_base}/NexusDesk.app/Contents/MacOS/nexusdesk"
else
  tar -C "${extract_dir}" -xzf "${package_path}"
  executable="${extract_dir}/${package_base}/bin/nexusdesk"
fi

if [[ ! -x "${executable}" ]]; then
  echo "Packaged executable is missing or not executable: ${executable}" >&2
  exit 1
fi

version_output="$("${executable}" --version)"
if [[ "${version_output}" != *"${version}"* ]]; then
  echo "Packaged executable --version did not include expected version ${version}." >&2
  echo "${version_output}" >&2
  exit 1
fi

smoke_output="$("${executable}" --smoke-check "${workspace_dir}")"
required_checks=(
  "workspace-open"
  "file-preview"
  "workspace-search"
  "edit-save-revert"
  "assistant-settings"
  "dataset-profile"
  "artifact-write-read"
  "diagnostics-export"
)
for check_name in "${required_checks[@]}"; do
  if [[ "${smoke_output}" != *"\"name\": \"${check_name}\""* || "${smoke_output}" != *"\"status\": \"ok\""* ]]; then
    echo "Packaged app smoke check missing or failed: ${check_name}" >&2
    echo "${smoke_output}" >&2
    exit 1
  fi
done

echo "${platform} package smoke passed."
echo "Package artifact: ${package_path}"
echo "Packaged executable: ${executable}"
echo "Installed app smoke checks: ${required_checks[*]}"
