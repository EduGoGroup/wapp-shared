#!/usr/bin/env bash

set -euo pipefail

CHANGELOG_FILE=""
VERSION=""

usage() {
    echo "Uso: $(basename "$0") --file <CHANGELOG.md> --version <vX.Y.Z|X.Y.Z>" >&2
}

while [[ $# -gt 0 ]]; do
    case "$1" in
        --file)
            CHANGELOG_FILE="$2"
            shift 2
            ;;
        --version)
            VERSION="$2"
            shift 2
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        *)
            echo "Argumento no soportado: $1" >&2
            usage
            exit 1
            ;;
    esac
done

if [[ -z "$CHANGELOG_FILE" || -z "$VERSION" ]]; then
    usage
    exit 1
fi

if [[ ! -f "$CHANGELOG_FILE" ]]; then
    echo "No existe el changelog: $CHANGELOG_FILE" >&2
    exit 1
fi

VERSION_NO_V="${VERSION#v}"
START_LINE=$(grep -n "^## \[$VERSION_NO_V\]" "$CHANGELOG_FILE" | head -n 1 | cut -d: -f1 || true)

if [[ -z "$START_LINE" ]]; then
    echo "No se encontro la version $VERSION_NO_V en $CHANGELOG_FILE" >&2
    exit 1
fi

END_LINE=$(awk -v start="$START_LINE" 'NR > start && /^## \[/ { print NR - 1; exit } END { if (NR >= start) print NR }' "$CHANGELOG_FILE")
sed -n "${START_LINE},${END_LINE}p" "$CHANGELOG_FILE"
