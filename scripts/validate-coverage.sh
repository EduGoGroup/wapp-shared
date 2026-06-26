#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
THRESHOLDS_FILE="$PROJECT_ROOT/.coverage-thresholds.yml"
MODULE_SCRIPT="$SCRIPT_DIR/list-modules.sh"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo "Validacion de Umbrales de Coverage"
echo ""

if [[ ! -f "$THRESHOLDS_FILE" ]]; then
    echo -e "${RED}Archivo de umbrales no encontrado: $THRESHOLDS_FILE${NC}"
    exit 1
fi

get_threshold() {
    local module="$1"
    awk -v module="$module" '
        function ltrim(s) { sub(/^[ \t\r\n]+/, "", s); return s }
        function rtrim(s) { sub(/[ \t\r\n]+$/, "", s); return s }
        function trim(s) { return rtrim(ltrim(s)) }
        /^[^[:space:]#].*:$/ {
            key = $0
            sub(/:.*/, "", key)
            current = trim(key)
        }
        current == module && /threshold:/ {
            value = $0
            sub(/.*threshold:[[:space:]]*/, "", value)
            value = trim(value)
            print value
            exit
        }
    ' "$THRESHOLDS_FILE"
}

validate_module() {
    local module="$1"
    local module_path="$PROJECT_ROOT/$module"
    local threshold coverage mod_download_log test_log meets diff

    threshold=$(get_threshold "$module")
    if [[ -z "$threshold" ]]; then
        threshold=$(get_threshold "global")
        threshold=$(awk '/default_threshold:/ { print $2; exit }' "$THRESHOLDS_FILE")
        echo -e "${YELLOW}$module: sin umbral explicito, usando default ${threshold}%${NC}"
    fi

    mod_download_log=$(mktemp)
    test_log=$(mktemp)

    (
        cd "$module_path"
        if ! go mod download >"$mod_download_log" 2>&1; then
            echo -e "${RED}$module: fallo go mod download${NC}"
            sed -n '1,80p' "$mod_download_log" | sed 's/^/  /'
            exit 10
        fi

        rm -f coverage.out
        if ! go test -short ./... -coverprofile=coverage.out -covermode=atomic >"$test_log" 2>&1; then
            echo -e "${RED}$module: tests fallan${NC}"
            sed -n '1,120p' "$test_log" | sed 's/^/  /'
            exit 11
        fi

        if [[ ! -f coverage.out ]]; then
            echo -e "${YELLOW}$module: sin archivo de coverage${NC}"
            exit 12
        fi

        coverage=$(go tool cover -func=coverage.out | tail -1 | awk '{print $NF}' | sed 's/%//')
        meets=$(echo "$coverage >= $threshold" | bc -l)

        if [[ "$meets" -eq 1 ]]; then
            diff=$(echo "$coverage - $threshold" | bc -l | awk '{printf "%.1f", $0}')
            echo -e "${GREEN}$module: ${coverage}% (umbral: ${threshold}%, +${diff}%)${NC}"
            exit 0
        fi

        diff=$(echo "$threshold - $coverage" | bc -l | awk '{printf "%.1f", $0}')
        echo -e "${RED}$module: ${coverage}% (umbral: ${threshold}%, -${diff}%)${NC}"
        exit 1
    )
    local result=$?

    rm -f "$mod_download_log" "$test_log" "$module_path/coverage.out"
    return "$result"
}

total=0
passed=0
failed=0

while IFS= read -r module; do
    [[ -z "$module" ]] && continue
    if [[ -f "$PROJECT_ROOT/$module/go.mod" ]]; then
        total=$((total + 1))
        if validate_module "$module"; then
            passed=$((passed + 1))
        else
            failed=$((failed + 1))
        fi
    fi
done < <("$MODULE_SCRIPT" --set coverage-validation)

echo ""
echo "Resumen"
echo "Total de modulos: $total"
echo -e "${GREEN}Pasaron: $passed${NC}"
echo -e "${RED}Fallaron: $failed${NC}"

if [[ "$failed" -gt 0 ]]; then
    exit 1
fi
