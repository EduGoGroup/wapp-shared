#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
MODULE_SCRIPT="$SCRIPT_DIR/list-modules.sh"
OUTPUT_DIR="$PROJECT_ROOT/docs/cicd/coverage-analysis"
OUTPUT_FILE="$OUTPUT_DIR/coverage-report-$(date +%Y%m%d).md"
CURRENT_DATE=$(date '+%Y-%m-%d %H:%M')

mkdir -p "$OUTPUT_DIR"

cat > "$OUTPUT_FILE" <<HEADER
# Reporte de Cobertura - wapp-shared

**Fecha:** $CURRENT_DATE
**Generado por:** analyze-coverage.sh

---

## Resumen Ejecutivo

| Modulo | Coverage | Estado | Prioridad |
|--------|----------|--------|-----------|
HEADER

evaluate_status() {
    local coverage="$1"

    if (( $(echo "$coverage >= 80" | bc -l) )); then
        echo "Excelente|Baja"
    elif (( $(echo "$coverage >= 60" | bc -l) )); then
        echo "Bueno|Baja"
    elif (( $(echo "$coverage >= 40" | bc -l) )); then
        echo "Aceptable|Media"
    elif (( $(echo "$coverage >= 20" | bc -l) )); then
        echo "Bajo|Alta"
    else
        echo "Critico|Critica"
    fi
}

analyze_module() {
    local module="$1"
    local module_path="$PROJECT_ROOT/$module"
    local log_file

    if [[ ! -d "$module_path" || ! -f "$module_path/go.mod" ]]; then
        echo "| $module | N/A | Modulo no encontrado | - |" >> "$OUTPUT_FILE"
        return
    fi

    log_file=$(mktemp)
    (
        cd "$module_path"
        rm -f coverage.out
        if go test -short ./... -coverprofile=coverage.out -covermode=atomic >"$log_file" 2>&1; then
            if [[ -f coverage.out ]]; then
                coverage=$(go tool cover -func=coverage.out | tail -1 | awk '{print $NF}' | sed 's/%//')
                IFS='|' read -r status priority < <(evaluate_status "$coverage")
                echo "| $module | ${coverage}% | $status | $priority |" >> "$OUTPUT_FILE"
                echo "" >> "$OUTPUT_FILE"
                echo "### $module (${coverage}%)" >> "$OUTPUT_FILE"
                echo "" >> "$OUTPUT_FILE"
                echo '```text' >> "$OUTPUT_FILE"
                go tool cover -func=coverage.out >> "$OUTPUT_FILE"
                echo '```' >> "$OUTPUT_FILE"
                echo "" >> "$OUTPUT_FILE"
            else
                echo "| $module | N/A | Sin archivo de coverage | - |" >> "$OUTPUT_FILE"
            fi
        else
            echo "| $module | ERROR | Tests fallan | - |" >> "$OUTPUT_FILE"
            echo "" >> "$OUTPUT_FILE"
            echo "### $module" >> "$OUTPUT_FILE"
            echo "" >> "$OUTPUT_FILE"
            echo '```text' >> "$OUTPUT_FILE"
            sed -n '1,80p' "$log_file" >> "$OUTPUT_FILE"
            echo '```' >> "$OUTPUT_FILE"
            echo "" >> "$OUTPUT_FILE"
        fi
        rm -f coverage.out "$log_file"
    )
}

while IFS= read -r module; do
    [[ -z "$module" ]] && continue
    analyze_module "$module"
done < <("$MODULE_SCRIPT" --set coverage-validation)

echo "Reporte generado: $OUTPUT_FILE"
