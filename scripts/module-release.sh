#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"
MODULE=""
VERSION=""
REMOTE="${REMOTE:-origin}"
DRY_RUN=0

usage() {
    cat <<USAGE >&2
Uso: $(basename "$0") --module <ruta/modulo> --version <vX.Y.Z> [--dry-run]
USAGE
}

run_cmd() {
    if [[ "$DRY_RUN" -eq 1 ]]; then
        printf '[dry-run]'
        printf ' %q' "$@"
        printf '\n'
        return 0
    fi

    "$@"
}

validate_semver() {
    [[ "$1" =~ ^v[0-9]+\.[0-9]+\.[0-9]+([-.][A-Za-z0-9.]+)?$ ]]
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
TAG="$MODULE/$VERSION"

if [[ ! -d "$MODULE_DIR" || ! -f "$MODULE_DIR/go.mod" ]]; then
    echo "Modulo invalido o sin go.mod: $MODULE" >&2
    exit 1
fi

if [[ ! -f "$CHANGELOG_FILE" ]]; then
    echo "No existe CHANGELOG.md en $MODULE" >&2
    exit 1
fi

if ! grep -q "^## \[$VERSION_NO_V\]" "$CHANGELOG_FILE"; then
    echo "Falta la seccion ## [$VERSION_NO_V] en $CHANGELOG_FILE. Ejecuta make changelog VERSION=$VERSION primero." >&2
    exit 1
fi

if [[ -n "$(git status --porcelain)" ]]; then
    if [[ "$DRY_RUN" -eq 1 ]]; then
        echo "Repositorio con cambios sin confirmar. Continuando solo por dry-run de $TAG."
    else
        echo "El repositorio tiene cambios sin confirmar. Haz commit antes de liberar $TAG." >&2
        exit 1
    fi
fi

if git rev-parse -q --verify "refs/tags/$TAG" >/dev/null; then
    echo "El tag $TAG ya existe localmente." >&2
    exit 1
fi

if git ls-remote --exit-code --tags "$REMOTE" "refs/tags/$TAG" >/dev/null 2>&1; then
    echo "El tag $TAG ya existe en remoto ($REMOTE)." >&2
    exit 1
fi

run_cmd git tag "$TAG"
run_cmd git push "$REMOTE" "$TAG"

echo "Tag publicado: $TAG"
echo "El workflow .github/workflows/release.yml generara el GitHub release usando $CHANGELOG_FILE."
