#!/usr/bin/env bash
# Builds the Bridge Talk flatpak for Linux (FR-810). Run from the repository root:
#
#   bash build_flatpak.sh
#
# Flow: install the flatpak tooling where it is missing, add flathub, install the GNOME runtime and
# the golang and node SDK extensions, write the desktop entry, the metainfo and the manifest, build
# the front end and then the application inside the sandbox, install it for the account running this
# and export one distributable bundle.
#
# PigeonPost's build_flatpak.sh is the reference for the build. The GNOME runtime is used because
# Wails renders through webkit2gtk-4.1, which the freedesktop runtime does not carry; that link is a
# cgo build, which is also what the audio output needs on Linux.
#
# The sandbox's grants are FR-813's, taken from o7 Debrief, whose flatpak watched a real session of
# the game under Proton. tests/structural/flatpak_test.go holds GRANTS below to exactly that list and
# APP_ID to the product's own id, so neither can drift from the requirement in silence.
#
# The machine voices' files, ONNX Runtime for Linux among them, are fetched inside the sandbox by
# tools/models, which checks each against the list's hash, then installed beside the executable. The
# repository's own models folder is skipped: it may hold another platform's runtime. The model is
# about 310 MB, so the bundle is that much larger than the application alone.
#
# Every generated file is written with printf rather than a here-document.
#
# Outputs: BridgeTalk.flatpak and a user install of the application id.
set -euo pipefail

APP_ID="uk.codecrafter.BridgeTalk"
BIN_NAME="BridgeTalk"
APP_NAME="Bridge Talk"
APP_SUMMARY="A ship's voice for Elite Dangerous"
HOMEPAGE="https://ernster.dev/BridgeTalk/"
PROJECT_LICENSE="GPL-3.0-only"
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
# MODELS_DIR is the folder the machine voices' files are read from, beside the executable; its name is
# voicefiles.Folder, which tests/structural/flatpak_test.go holds this to.
MODELS_DIR="models"
VERSION="$(tr -d '[:space:]' < VERSION)"
RELEASE_DATE="$(date +%F)"

# GRANTS are the sandbox's permissions (FR-813), one per line, each once. No network is granted: the
# application makes no request (NFR-S-1).
GRANTS=(
    --share=ipc
    --socket=wayland
    --socket=fallback-x11
    --device=dri
    --socket=pulseaudio
    --filesystem=home
    --filesystem=~/.var/app/com.valvesoftware.Steam:ro
    --filesystem=xdg-config/autostart:create
    --talk-name=org.kde.StatusNotifierWatcher
)

section() { printf '\n\033[1m== %s ==\033[0m\n' "$1"; }

install_if_missing() {
    local tool="$1"
    command -v "$tool" > /dev/null 2>&1 && return 0
    section "Installing missing tool: $tool"
    if command -v apt-get > /dev/null 2>&1; then sudo apt-get install -y "$tool"
    elif command -v dnf > /dev/null 2>&1; then sudo dnf install -y "$tool"
    elif command -v pacman > /dev/null 2>&1; then sudo pacman -S --noconfirm "$tool"
    elif command -v zypper > /dev/null 2>&1; then sudo zypper install -y "$tool"
    else echo "error: install $tool with your package manager and run this again" >&2; exit 1
    fi
}

# write appends each argument to a file as one line.
write() {
    local file="$1"
    shift
    printf '%s\n' "$@" >> "$file"
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

section "Writing packaging files"
rm -rf "${PACKAGING_DIR}"
mkdir -p "${PACKAGING_DIR}"

DESKTOP="${PACKAGING_DIR}/${APP_ID}.desktop"
write "$DESKTOP" \
    "[Desktop Entry]" \
    "Name=${APP_NAME}" \
    "Comment=${APP_SUMMARY}" \
    "Exec=${BIN_NAME}" \
    "Icon=${APP_ID}" \
    "Terminal=false" \
    "Type=Application" \
    "Categories=Game;Utility;"

METAINFO="${PACKAGING_DIR}/${APP_ID}.metainfo.xml"
write "$METAINFO" \
    '<?xml version="1.0" encoding="UTF-8"?>' \
    '<component type="desktop-application">' \
    "  <id>${APP_ID}</id>" \
    "  <name>${APP_NAME}</name>" \
    "  <summary>${APP_SUMMARY}</summary>" \
    "  <metadata_license>CC0-1.0</metadata_license>" \
    "  <project_license>${PROJECT_LICENSE}</project_license>" \
    "  <description>" \
    "    <p>Bridge Talk watches the game's journal and status file while you play and speaks for the moments worth saying, in recordings you supply.</p>" \
    "  </description>" \
    "  <launchable type=\"desktop-id\">${APP_ID}.desktop</launchable>" \
    "  <url type=\"homepage\">${HOMEPAGE}</url>" \
    '  <content_rating type="oars-1.1"/>' \
    "  <releases>" \
    "    <release version=\"${VERSION}\" date=\"${RELEASE_DATE}\"/>" \
    "  </releases>" \
    "</component>"

section "Writing manifest"
rm -f "${MANIFEST}"
write "$MANIFEST" \
    "app-id: ${APP_ID}" \
    "runtime: ${RUNTIME}" \
    "runtime-version: '${RUNTIME_VERSION}'" \
    "sdk: ${SDK}" \
    "sdk-extensions:" \
    "  - ${GOLANG_EXT}" \
    "  - ${NODE_EXT}" \
    "command: ${BIN_NAME}" \
    "finish-args:"
for grant in "${GRANTS[@]}"; do
    write "$MANIFEST" "  - ${grant}"
done
write "$MANIFEST" \
    "build-options:" \
    "  append-path: /usr/lib/sdk/golang/bin:/usr/lib/sdk/node22/bin" \
    "  build-args:" \
    "    - --share=network" \
    "  env:" \
    "    CGO_ENABLED: '1'" \
    "    GOPATH: /run/build/${BIN_NAME}/gopath" \
    "    GOCACHE: /run/build/${BIN_NAME}/gocache" \
    "    GOFLAGS: -buildvcs=false" \
    "    npm_config_cache: /run/build/${BIN_NAME}/npm-cache" \
    "modules:" \
    "  - name: ${BIN_NAME}" \
    "    buildsystem: simple" \
    "    build-commands:" \
    "      - cd frontend && npm install --no-audit --no-fund && npm run build" \
    "      - go build -tags desktop,production,webkit2_41 -ldflags '-s -w -X main.appVersion=${VERSION}' -o ${BIN_NAME} ." \
    "      - install -Dm755 ${BIN_NAME} /app/bin/${BIN_NAME}" \
    "      - go run ./tools/models" \
    "      - install -Dm644 -t /app/bin/${MODELS_DIR} ${MODELS_DIR}/*" \
    "      - go run ./tools/linuxicons -prefix /app" \
    "      - chmod -R u+w /run/build/${BIN_NAME}/gopath /run/build/${BIN_NAME}/gocache 2>/dev/null || true" \
    "      - install -Dm644 ${DESKTOP} /app/share/applications/${APP_ID}.desktop" \
    "      - install -Dm644 ${METAINFO} /app/share/metainfo/${APP_ID}.metainfo.xml" \
    "    sources:" \
    "      - type: dir" \
    "        path: ." \
    "        skip:" \
    "          - .git" \
    "          - ${BUILD_DIR}" \
    "          - ${REPO_DIR}" \
    "          - .flatpak-builder" \
    "          - ${BUNDLE}" \
    "          - build" \
    "          - ${MODELS_DIR}" \
    "          - frontend/node_modules" \
    "          - frontend/dist"

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
