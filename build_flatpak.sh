#!/usr/bin/env bash
# Builds the WebVigil Flatpak for Linux. Run from the repo root:
#
#   bash build_flatpak.sh
#
# Flow: install flatpak tooling if missing, add flathub, install the GNOME runtime and the
# golang and node SDK extensions, self-generate the desktop file, metainfo and manifest,
# build the app inside the sandbox (npm front end, then a CGO Go build against the runtime's
# webkit2gtk-4.1), install it for the current user and export a distributable bundle.
#
# The GNOME runtime is required because Wails v2 renders through webkit2gtk-4.1, which the
# freedesktop runtime does not carry. The Go build therefore uses -tags webkit2_41.
#
# Outputs: webvigil.flatpak (installable anywhere) and a user install of ${APP_ID}.
set -euo pipefail

APP_ID="uk.codecrafter.WebVigil"
BIN_NAME="webvigil"
APP_NAME="WebVigil"
APP_SUMMARY="Watch the web. Know what changes."
HOMEPAGE="https://ernster.dev/WebVigil/"
RUNTIME="org.gnome.Platform"
SDK="org.gnome.Sdk"
RUNTIME_VERSION="50"
SDK_EXT_VERSION="25.08"   # the freedesktop base of GNOME 50; extensions pair with it
GOLANG_EXT="org.freedesktop.Sdk.Extension.golang"
NODE_EXT="org.freedesktop.Sdk.Extension.node22"
BUILD_DIR=".flatpak-build"
REPO_DIR=".flatpak-repo"
BUNDLE="${BIN_NAME}.flatpak"
MANIFEST="${APP_ID}.yml"
PACKAGING_DIR="packaging"
# The same list as hicolorSizes in tools/genicons, which writes build/linux/icons/webvigil_<size>.png.
HICOLOR_SIZES="16 24 32 48 64 128 256 512"
VERSION="$(tr -d '[:space:]' < VERSION)"
RELEASE_DATE="$(date +%F)"

section() { printf '\n\033[1m== %s ==\033[0m\n' "$1"; }

install_if_missing() {
    local tool="$1"
    command -v "$tool" > /dev/null 2>&1 && return 0
    section "Installing missing tool: $tool"
    if command -v apt-get > /dev/null 2>&1; then sudo apt-get install -y "$tool"
    elif command -v dnf > /dev/null 2>&1; then sudo dnf install -y "$tool"
    elif command -v pacman > /dev/null 2>&1; then sudo pacman -S --noconfirm "$tool"
    elif command -v zypper > /dev/null 2>&1; then sudo zypper install -y "$tool"
    else echo "error: install $tool with your package manager and re-run" >&2; exit 1
    fi
}

section "Tooling"
install_if_missing flatpak
install_if_missing flatpak-builder

section "Flathub remote and runtimes"
flatpak remote-add --if-not-exists --user flathub https://dl.flathub.org/repo/flathub.flatpakrepo
flatpak install --user --noninteractive flathub \
    "${RUNTIME}//${RUNTIME_VERSION}" \
    "${SDK}//${RUNTIME_VERSION}" \
    "${GOLANG_EXT}//${SDK_EXT_VERSION}" \
    "${NODE_EXT}//${SDK_EXT_VERSION}"

section "Release repository"
# The repository that publishes the releases, read from this checkout's own git remote so the
# source names no account (the same rule as build.ps1). It is read here on the host because the
# sandbox is given the tree without .git. Without one the update check reports every check as
# unreachable rather than asking the wrong project.
ORIGIN="$(git remote get-url origin 2> /dev/null || true)"
RELEASE_REPO="$(printf '%s' "${ORIGIN}" | sed -nE 's#.*github\.com[:/]([^/]+/[^/]+)$#\1#p' | sed -E 's#\.git$##')"
if [ -n "${RELEASE_REPO}" ]; then
    echo "Release repository: ${RELEASE_REPO}"
else
    echo "warning: no GitHub origin remote: this build cannot check for updates" >&2
fi

section "Single-instance bus name"
# Wails v2 keeps one WebVigil running on Linux by owning a session-bus name derived from the
# SingleInstanceLock id in main.go (wails internal/frontend/desktop/linux/single_instance.go):
# org.wails_app_<id with '-' and '.' as '_'>.SingleInstance. The sandbox may own only names it
# is granted, so the name is read from main.go rather than restated; it cannot drift from it.
INSTANCE_ID="$(sed -nE 's/^[[:space:]]*instanceID[[:space:]]*=[[:space:]]*"([^"]+)".*/\1/p' main.go)"
[ -n "${INSTANCE_ID}" ] || { echo "error: instanceID not found in main.go" >&2; exit 1; }
SINGLE_INSTANCE_NAME="org.wails_app_$(printf '%s' "${INSTANCE_ID}" | tr -- '-.' '__').SingleInstance"
echo "${SINGLE_INSTANCE_NAME}"

section "Writing packaging files"
mkdir -p "${PACKAGING_DIR}"

printf '%s\n' \
    "[Desktop Entry]" \
    "Name=${APP_NAME}" \
    "Comment=${APP_SUMMARY}" \
    "Exec=${BIN_NAME}" \
    "Icon=${APP_ID}" \
    "Terminal=false" \
    "Type=Application" \
    "Categories=Network;Utility;" \
    > "${PACKAGING_DIR}/${APP_ID}.desktop"

printf '%s\n' \
    '<?xml version="1.0" encoding="UTF-8"?>' \
    '<component type="desktop-application">' \
    "  <id>${APP_ID}</id>" \
    "  <name>${APP_NAME}</name>" \
    "  <summary>${APP_SUMMARY}</summary>" \
    "  <metadata_license>CC0-1.0</metadata_license>" \
    "  <project_license>GPL-3.0-only</project_license>" \
    "  <description>" \
    "    <p>" \
    "      WebVigil checks the web pages you choose on a schedule, compares what it reads against" \
    "      the last version it accepted and tells you when the meaningful content changes. When a" \
    "      check fails it says why. Everything it records stays on your computer." \
    "    </p>" \
    "  </description>" \
    "  <launchable type=\"desktop-id\">${APP_ID}.desktop</launchable>" \
    "  <url type=\"homepage\">${HOMEPAGE}</url>" \
    '  <content_rating type="oars-1.1"/>' \
    "  <releases>" \
    "    <release version=\"${VERSION}\" date=\"${RELEASE_DATE}\"/>" \
    "  </releases>" \
    "</component>" \
    > "${PACKAGING_DIR}/${APP_ID}.metainfo.xml"

section "Writing manifest"
ICON_INSTALL_CMDS=()
for size in ${HICOLOR_SIZES}; do
    ICON_INSTALL_CMDS+=("      - install -Dm644 build/linux/icons/${BIN_NAME}_${size}.png /app/share/icons/hicolor/${size}x${size}/apps/${APP_ID}.png")
done

printf '%s\n' \
    "app-id: ${APP_ID}" \
    "runtime: ${RUNTIME}" \
    "runtime-version: '${RUNTIME_VERSION}'" \
    "sdk: ${SDK}" \
    "sdk-extensions:" \
    "  - ${GOLANG_EXT}" \
    "  - ${NODE_EXT}" \
    "command: ${BIN_NAME}" \
    "finish-args:" \
    "  - --share=ipc" \
    "  - --socket=fallback-x11" \
    "  - --socket=wayland" \
    "  - --device=dri" \
    "  # The page checks and the update check both need the network; without it every check fails." \
    "  - --share=network" \
    "  # Import and Export open the toolkit's own file dialog inside the sandbox." \
    "  - --filesystem=home" \
    "  - --talk-name=org.freedesktop.Notifications" \
    "  # The Wails single-instance lock; see the Single-instance bus name step in build_flatpak.sh." \
    "  - --own-name=${SINGLE_INSTANCE_NAME}" \
    "build-options:" \
    "  append-path: /usr/lib/sdk/golang/bin:/usr/lib/sdk/node22/bin" \
    "  # build-args grants the BUILD network access to fetch modules; it grants the app nothing." \
    "  build-args:" \
    "    - --share=network" \
    "  env:" \
    "    GOPATH: /run/build/${BIN_NAME}/gopath" \
    "    GOCACHE: /run/build/${BIN_NAME}/gocache" \
    "    GOFLAGS: -buildvcs=false" \
    "    npm_config_cache: /run/build/${BIN_NAME}/npm-cache" \
    "modules:" \
    "  - name: ${BIN_NAME}" \
    "    buildsystem: simple" \
    "    build-commands:" \
    "      # npm install, not npm ci, to match wails.json frontend:install, which every other build runs." \
    "      - cd frontend && npm install --no-audit --no-fund && npm run build" \
    "      - go run ./tools/genicons" \
    "      - go build -tags desktop,production,webkit2_41 -ldflags \"-s -w -X main.appVersion=${VERSION} -X main.releaseRepo=${RELEASE_REPO}\" -o ${BIN_NAME} ." \
    "      - install -Dm755 ${BIN_NAME} /app/bin/${BIN_NAME}" \
    "      - chmod -R u+w gopath gocache 2>/dev/null || true" \
    "      - install -Dm644 ${PACKAGING_DIR}/${APP_ID}.desktop /app/share/applications/${APP_ID}.desktop" \
    "      - install -Dm644 ${PACKAGING_DIR}/${APP_ID}.metainfo.xml /app/share/metainfo/${APP_ID}.metainfo.xml" \
    "${ICON_INSTALL_CMDS[@]}" \
    "    sources:" \
    "      - type: dir" \
    "        path: ." \
    "        skip:" \
    "          - .git" \
    "          - ${BUILD_DIR}" \
    "          - ${REPO_DIR}" \
    "          - .flatpak-builder" \
    "          - ${BUNDLE}" \
    "          - frontend/node_modules" \
    > "${MANIFEST}"

section "Building ${APP_NAME} ${VERSION} with flatpak-builder"
flatpak-builder --user --install --force-clean \
    --install-deps-from=flathub \
    --repo="${REPO_DIR}" \
    "${BUILD_DIR}" "${MANIFEST}"

section "Exporting bundle"
flatpak build-bundle \
    --runtime-repo=https://dl.flathub.org/repo/flathub.flatpakrepo \
    "${REPO_DIR}" "${BUNDLE}" "${APP_ID}"

section "Done"
echo "Installed for the current user: flatpak run ${APP_ID}"
echo "Distributable bundle: ${BUNDLE}"
