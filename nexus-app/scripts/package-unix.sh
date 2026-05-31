#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
APP_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
REPO_ROOT="$(cd "${APP_ROOT}/.." && pwd)"

output_dir="${APP_ROOT}/dist"
version="${NEXUSDESK_VERSION:-0.0.0-ci}"
commit="${GITHUB_SHA:-}"
build_date="${NEXUSDESK_BUILD_DATE:-}"
macos_sign_identity="${NEXUSDESK_MACOS_CODESIGN_IDENTITY:-}"
macos_notary_profile="${NEXUSDESK_MACOS_NOTARY_PROFILE:-}"

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
    -c|--commit)
      commit="$2"
      shift 2
      ;;
    -d|--build-date)
      build_date="$2"
      shift 2
      ;;
    --macos-sign-identity)
      macos_sign_identity="$2"
      shift 2
      ;;
    --macos-notary-profile)
      macos_notary_profile="$2"
      shift 2
      ;;
    *)
      echo "Unknown argument: $1" >&2
      exit 2
      ;;
  esac
done

if [[ -z "${commit}" ]]; then
  commit="$(git -C "${REPO_ROOT}" rev-parse --short HEAD)"
fi
commit="${commit:0:12}"
if [[ -z "${build_date}" ]]; then
  build_date="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
fi

export CGO_ENABLED=1
export GOFLAGS="-mod=readonly"

goos="$(go env GOOS)"
goarch="$(go env GOARCH)"
case "${goos}" in
  darwin)
    platform="macos"
    ;;
  linux)
    platform="linux"
    ;;
  *)
    echo "package-unix.sh supports linux and macOS, got ${goos}." >&2
    exit 2
    ;;
esac

safe_version="$(printf '%s' "${version}" | sed 's/[^0-9A-Za-z._+-]/-/g')"
package_base="nexusdesk-${platform}-${goarch}-${safe_version}"
mkdir -p "${output_dir}"
output_root="$(cd "${output_dir}" && pwd -P)"
staging="${output_root}/${package_base}"
if [[ "${platform}" == "macos" ]]; then
  package_path="${output_root}/${package_base}.zip"
  notary_upload_path="${output_root}/${package_base}-notary-upload.zip"
else
  package_path="${output_root}/${package_base}.tar.gz"
  notary_upload_path=""
fi
manifest_path="${output_root}/${package_base}-manifest.json"
sbom_path="${output_root}/${package_base}-sbom.json"
provenance_path="${output_root}/${package_base}-provenance.json"

assert_under_output() {
  local target="$1"
  case "${target}" in
    "${output_root}"/*) ;;
    *)
      echo "Refusing to remove path outside output directory: ${target}" >&2
      exit 2
      ;;
  esac
}

for path in "${staging}" "${package_path}" "${notary_upload_path}" "${manifest_path}" "${sbom_path}" "${provenance_path}"; do
  if [[ -z "${path}" ]]; then
    continue
  fi
  if [[ -e "${path}" ]]; then
    assert_under_output "${path}"
    rm -rf "${path}"
  fi
done
mkdir -p "${staging}"

ldflags="-X nexusdesk/internal/buildinfo.Version=${version} -X nexusdesk/internal/buildinfo.Commit=${commit} -X nexusdesk/internal/buildinfo.BuildDate=${build_date}"

cd "${APP_ROOT}"
if [[ "${platform}" == "macos" ]]; then
  app_dir="${staging}/NexusDesk.app"
  mkdir -p "${app_dir}/Contents/MacOS" "${app_dir}/Contents/Resources"
  go build -ldflags "${ldflags}" -o "${app_dir}/Contents/MacOS/nexusdesk" .
  chmod 0755 "${app_dir}/Contents/MacOS/nexusdesk"
  cat > "${app_dir}/Contents/Info.plist" <<PLIST
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>CFBundleDevelopmentRegion</key>
  <string>en</string>
  <key>CFBundleDisplayName</key>
  <string>NexusDesk</string>
  <key>CFBundleExecutable</key>
  <string>nexusdesk</string>
  <key>CFBundleIdentifier</key>
  <string>com.nexusdesk.app</string>
  <key>CFBundleInfoDictionaryVersion</key>
  <string>6.0</string>
  <key>CFBundleName</key>
  <string>NexusDesk</string>
  <key>CFBundlePackageType</key>
  <string>APPL</string>
  <key>CFBundleShortVersionString</key>
  <string>${version}</string>
  <key>CFBundleVersion</key>
  <string>${commit}</string>
  <key>LSMinimumSystemVersion</key>
  <string>13.0</string>
  <key>NSHighResolutionCapable</key>
  <true/>
</dict>
</plist>
PLIST
  if [[ -f "${APP_ROOT}/internal/brand/assets/nexus-app-icon-transparent.png" ]]; then
    cp "${APP_ROOT}/internal/brand/assets/nexus-app-icon-transparent.png" "${app_dir}/Contents/Resources/nexusdesk.png"
  fi
  if [[ -n "${macos_sign_identity}" ]]; then
    codesign --force --timestamp --options runtime --sign "${macos_sign_identity}" "${app_dir}"
  fi
  if [[ -n "${macos_notary_profile}" ]]; then
    if [[ -z "${macos_sign_identity}" ]]; then
      echo "--macos-notary-profile requires --macos-sign-identity or NEXUSDESK_MACOS_CODESIGN_IDENTITY." >&2
      exit 2
    fi
    ditto -c -k --sequesterRsrc --keepParent "${app_dir}" "${notary_upload_path}"
    xcrun notarytool submit "${notary_upload_path}" --keychain-profile "${macos_notary_profile}" --wait
    xcrun stapler staple "${app_dir}"
    rm -f "${notary_upload_path}"
  fi
  cat > "${staging}/README.txt" <<README
NexusDesk macOS package
Version: ${version}
Commit: ${commit}
Build date: ${build_date}

Open NexusDesk.app after verifying the package manifest, SBOM, and provenance sidecars.
Signing identity: ${macos_sign_identity:-not signed}
Notarization profile: ${macos_notary_profile:-not notarized}

Public releases require Developer ID signing, notarization, and stapling before broad distribution.
README
else
  mkdir -p "${staging}/bin" "${staging}/share/applications" "${staging}/share/icons/hicolor/256x256/apps"
  go build -ldflags "${ldflags}" -o "${staging}/bin/nexusdesk" .
  chmod 0755 "${staging}/bin/nexusdesk"
  if [[ -f "${APP_ROOT}/internal/brand/assets/nexus-app-icon-transparent.png" ]]; then
    cp "${APP_ROOT}/internal/brand/assets/nexus-app-icon-transparent.png" "${staging}/share/icons/hicolor/256x256/apps/nexusdesk.png"
  fi
  cat > "${staging}/share/applications/nexusdesk.desktop" <<DESKTOP
[Desktop Entry]
Type=Application
Name=NexusDesk
Comment=Fyne-native local workspace and assistant workbench
Exec=nexusdesk
Icon=nexusdesk
Terminal=false
Categories=Development;Utility;
DESKTOP
  cat > "${staging}/README.txt" <<README
NexusDesk Linux package
Version: ${version}
Commit: ${commit}
Build date: ${build_date}

Install by placing bin/nexusdesk on PATH and copying share/applications plus share/icons into the matching XDG locations.
Verify the package manifest, SBOM, and provenance sidecars before installing.
README
fi

if [[ "${platform}" == "macos" ]]; then
  ditto -c -k --sequesterRsrc --keepParent "${staging}" "${package_path}"
else
  tar -C "${output_root}" -czf "${package_path}" "${package_base}"
fi

go run ./cmd/release-manifest \
  -artifact "${package_path}" \
  -output "${manifest_path}" \
  -platform "${platform}" \
  -version "${version}" \
  -commit "${commit}" \
  -build-date "${build_date}" \
  -repository "RCooLeR/NexusDesk" \
  -workflow "scripts/package-unix.sh" \
  -source-commit-full "${commit}"

for path in "${package_path}" "${manifest_path}" "${sbom_path}" "${provenance_path}"; do
  if [[ ! -s "${path}" ]]; then
    echo "Expected package output was not generated: ${path}" >&2
    exit 1
  fi
done

package_bytes="$(wc -c < "${package_path}" | tr -d '[:space:]')"
echo "Wrote ${platform} package: ${package_path}"
echo "Package bytes: ${package_bytes}"
echo "Evidence: ${manifest_path}, ${sbom_path}, ${provenance_path}"
