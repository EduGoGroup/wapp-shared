#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MANIFEST_FILE="$SCRIPT_DIR/module-manifest.tsv"
SET_NAME="all"
FORMAT="plain"

usage() {
    cat <<USAGE
Uso: $(basename "$0") [--set <nombre>] [--format plain|json]

Sets disponibles:
  all
  integration
  coverage-validation
  release
  level-0
  level-1
  level-2
  level-3
USAGE
}

while [[ $# -gt 0 ]]; do
    case "$1" in
        --set)
            SET_NAME="$2"
            shift 2
            ;;
        --format)
            FORMAT="$2"
            shift 2
            ;;
        --json)
            FORMAT="json"
            shift
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        *)
            echo "Argumento no soportado: $1" >&2
            usage >&2
            exit 1
            ;;
    esac
done

if [[ ! -f "$MANIFEST_FILE" ]]; then
    echo "No se encontro el manifest de modulos: $MANIFEST_FILE" >&2
    exit 1
fi

filter_modules() {
    awk -F'|' -v set_name="$SET_NAME" '
        /^[[:space:]]*#/ || NF < 4 { next }
        {
            module = $1
            level = $2
            integration = $3
            coverage = $4

            if (set_name == "all" || set_name == "release") {
                print module
            } else if (set_name == "integration" && integration == "true") {
                print module
            } else if (set_name == "coverage-validation" && coverage == "true") {
                print module
            } else if (set_name == ("level-" level)) {
                print module
            }
        }
    ' "$MANIFEST_FILE"
}

mapfile -t MODULES < <(filter_modules)

if [[ "$FORMAT" == "json" ]]; then
    printf '['
    for i in "${!MODULES[@]}"; do
        if [[ "$i" -gt 0 ]]; then
            printf ','
        fi
        printf '"%s"' "${MODULES[$i]}"
    done
    printf ']\n'
    exit 0
fi

printf '%s\n' "${MODULES[@]}"
