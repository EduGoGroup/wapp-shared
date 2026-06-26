#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"
MODULE=""
VERSION=""
RELEASE_DATE="$(date +%Y-%m-%d)"
DRY_RUN=0

usage() {
    cat <<USAGE >&2
Uso: $(basename "$0") --module <ruta/modulo> --version <vX.Y.Z> [--date YYYY-MM-DD] [--dry-run]
USAGE
}

trim_block() {
    awk '
        { lines[NR] = $0 }
        $0 ~ /[^[:space:]]/ {
            if (first == 0) {
                first = NR
            }
            last = NR
        }
        END {
            if (first == 0) {
                exit
            }
            for (i = first; i <= last; i++) {
                print lines[i]
            }
        }
    '
}

validate_semver() {
    [[ "$1" =~ ^v[0-9]+\.[0-9]+\.[0-9]+([-.][A-Za-z0-9.]+)?$ ]]
}

generate_changes() {
    local previous_tag
    previous_tag=$(git tag --list "$MODULE/v*" --sort=-version:refname | head -n 1 || true)

    if [[ -n "$previous_tag" ]]; then
        git log --no-merges --format='- %s (%h)' "$previous_tag..HEAD" -- "$MODULE"
    else
        git log --no-merges --format='- %s (%h)' HEAD -- "$MODULE"
    fi
}

while [[ $# -gt 0 ]]; do
    case "$1" in
        --module)
            MODULE="$2"
            shift 2
            ;;
        --version)
            VERSION="$2"
            shift 2
            ;;
        --date)
            RELEASE_DATE="$2"
            shift 2
            ;;
        --dry-run)
            DRY_RUN=1
            shift
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

if [[ -z "$MODULE" || -z "$VERSION" ]]; then
    usage
    exit 1
fi

if ! validate_semver "$VERSION"; then
    echo "Version invalida: $VERSION. Usa el formato vX.Y.Z" >&2
    exit 1
fi

MODULE_DIR="$ROOT_DIR/$MODULE"
CHANGELOG_FILE="$MODULE_DIR/CHANGELOG.md"
VERSION_NO_V="${VERSION#v}"

if [[ ! -d "$MODULE_DIR" || ! -f "$MODULE_DIR/go.mod" ]]; then
    echo "Modulo invalido o sin go.mod: $MODULE" >&2
    exit 1
fi

if [[ ! -f "$CHANGELOG_FILE" ]]; then
    echo "No existe CHANGELOG.md en $MODULE" >&2
    exit 1
fi

if grep -q "^## \[$VERSION_NO_V\]" "$CHANGELOG_FILE"; then
    echo "La version $VERSION_NO_V ya existe en $CHANGELOG_FILE"
    exit 0
fi

UNRELEASED_LINE=$(grep -n '^## \[Unreleased\]' "$CHANGELOG_FILE" | head -n 1 | cut -d: -f1 || true)
if [[ -z "$UNRELEASED_LINE" ]]; then
    echo "El changelog de $MODULE no contiene la seccion ## [Unreleased]" >&2
    exit 1
fi

NEXT_SECTION_LINE=$(awk -v start="$UNRELEASED_LINE" 'NR > start && /^## \[/ { print NR; exit }' "$CHANGELOG_FILE")
BODY_START=$((UNRELEASED_LINE + 1))
if [[ -n "$NEXT_SECTION_LINE" ]]; then
    BODY_END=$((NEXT_SECTION_LINE - 1))
    UNRELEASED_BODY=$(sed -n "${BODY_START},${BODY_END}p" "$CHANGELOG_FILE" | trim_block)
else
    UNRELEASED_BODY=$(tail -n +"$BODY_START" "$CHANGELOG_FILE" | trim_block)
fi

if [[ -z "$UNRELEASED_BODY" ]]; then
    GENERATED_CHANGES=$(generate_changes | trim_block)
    if [[ -n "$GENERATED_CHANGES" ]]; then
        UNRELEASED_BODY=$(printf '### Changed\n\n%s\n' "$GENERATED_CHANGES")
    else
        UNRELEASED_BODY=$'### Changed\n\n- Sin cambios registrados.'
    fi
fi

if ! printf '%s\n' "$UNRELEASED_BODY" | grep -q '^### '; then
    UNRELEASED_BODY=$(printf '### Changed\n\n%s\n' "$UNRELEASED_BODY")
fi

TMP_FILE=$(mktemp)
head -n "$UNRELEASED_LINE" "$CHANGELOG_FILE" > "$TMP_FILE"
printf '\n' >> "$TMP_FILE"
printf '## [%s] - %s\n\n' "$VERSION_NO_V" "$RELEASE_DATE" >> "$TMP_FILE"
printf '%s\n' "$UNRELEASED_BODY" >> "$TMP_FILE"
printf '\n' >> "$TMP_FILE"

if [[ -n "$NEXT_SECTION_LINE" ]]; then
    tail -n +"$NEXT_SECTION_LINE" "$CHANGELOG_FILE" >> "$TMP_FILE"
fi

if [[ "$DRY_RUN" -eq 1 ]]; then
    diff -u "$CHANGELOG_FILE" "$TMP_FILE" || true
    rm -f "$TMP_FILE"
    exit 0
fi

mv "$TMP_FILE" "$CHANGELOG_FILE"
echo "CHANGELOG actualizado: $CHANGELOG_FILE"
