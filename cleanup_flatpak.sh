#!/usr/bin/env bash
# Uninstalls the Bridge Talk flatpak and removes what build_flatpak.sh made (FR-810).
#
# Scoped to the flatpak alone. It does not touch build/, dist-installer/ or anything else another
# build path makes, so the build paths stay independent. It does not touch the user's recordings,
# settings or log either: those belong to the user, not to the build.
set -euo pipefail

APP_ID="uk.codecrafter.BridgeTalk"
BIN_NAME="BridgeTalk"

section() { printf '\n\033[1m== %s ==\033[0m\n' "$1"; }

section "Uninstalling ${APP_ID}"
if flatpak list --user --app --columns=application | grep -qx "${APP_ID}"; then
    flatpak uninstall --user -y "${APP_ID}"
    echo "  Uninstalled."
else
    echo "  Not installed, so nothing to uninstall."
fi

section "Removing what the flatpak build made"
rm -f "${BIN_NAME}.flatpak"
rm -rf .flatpak-build .flatpak-repo .flatpak-builder packaging
rm -f "${APP_ID}.yml"
echo "  Done."
