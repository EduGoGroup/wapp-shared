#!/usr/bin/env bash
#
# auto-release.sh - Automated module release script
#
# Detects modified CHANGELOGs, extracts versions, creates commits and tags,
# and pushes them to trigger GitHub Actions release workflow.
#
# Usage: auto-release.sh [OPTIONS] [MODULES...]
#

set -euo pipefail

# ============================================================================
# CONFIGURATION
# ============================================================================

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SCRIPT_NAME="$(basename "${BASH_SOURCE[0]}")"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
MAGENTA='\033[0;35m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m' # No Color

# Options
DRY_RUN=false
VERBOSE=false
PROCESS_ALL=false
AUTO_YES=false
REMOTE="origin"
SPECIFIED_MODULES=()

# State
declare -A MODULE_VERSIONS
declare -A MODULE_TAGS
declare -A MODULE_CHANGELOGS
MODULES_TO_PROCESS=()

# ============================================================================
# UTILITY FUNCTIONS
# ============================================================================

log_info() {
  echo -e "${BLUE}[INFO]${NC} $*" >&2
}

log_success() {
  echo -e "${GREEN}[SUCCESS]${NC} $*" >&2
}

log_warning() {
  echo -e "${YELLOW}[WARNING]${NC} $*" >&2
}

log_error() {
  echo -e "${RED}[ERROR]${NC} $*" >&2
}

log_verbose() {
  if [[ "$VERBOSE" == "true" ]]; then
    echo -e "${CYAN}[VERBOSE]${NC} $*" >&2
  fi
}

log_section() {
  echo "" >&2
  echo -e "${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}" >&2
  echo -e "${BOLD}$*${NC}" >&2
  echo -e "${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}" >&2
  echo "" >&2
}

fail() {
  log_error "$*"
  exit 1
}

run_cmd() {
  local cmd="$*"
  log_verbose "Ejecutando: $cmd"
  
  if [[ "$DRY_RUN" == "true" ]]; then
    log_warning "[DRY-RUN] Se ejecutaría: $cmd"
    return 0
  fi
  
  eval "$cmd"
}

confirm() {
  local prompt="$1"
  local default="${2:-n}"
  
  if [[ "$AUTO_YES" == "true" ]]; then
    log_verbose "Auto-confirmado (--yes)"
    return 0
  fi
  
  local yn
  if [[ "$default" == "y" ]]; then
    read -r -p "$prompt (Y/n): " yn
    yn=${yn:-y}
  else
    read -r -p "$prompt (y/N): " yn
    yn=${yn:-n}
  fi
  
  case "$yn" in
    [Yy]*) return 0 ;;
    *) return 1 ;;
  esac
}

# ============================================================================
# VALIDATION FUNCTIONS
# ============================================================================

check_git_repo() {
  log_verbose "Verificando repositorio Git..."
  if ! git rev-parse --git-dir > /dev/null 2>&1; then
    fail "No estás en un repositorio Git"
  fi
  log_verbose "✓ Repositorio Git válido"
}

check_remote() {
  log_verbose "Verificando remote '$REMOTE'..."
  if ! git remote get-url "$REMOTE" > /dev/null 2>&1; then
    fail "Remote '$REMOTE' no existe"
  fi
  log_verbose "✓ Remote '$REMOTE' existe"
}

validate_version_format() {
  local version="$1"
  if [[ ! "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    return 1
  fi
  return 0
}

check_tag_exists() {
  local tag="$1"
  log_verbose "Verificando si el tag '$tag' existe..."
  if git tag -l "$tag" | grep -q "^$tag$"; then
    log_verbose "✗ Tag '$tag' ya existe"
    return 0
  fi
  log_verbose "✓ Tag '$tag' no existe (OK)"
  return 1
}

# ============================================================================
# CHANGELOG DETECTION AND PARSING
# ============================================================================

detect_modified_changelogs() {
  log_verbose "Detectando CHANGELOGs modificados..."
  
  # Detect both unstaged and staged changes
  local unstaged_files staged_files modified_files
  unstaged_files=$(git diff --name-only 2>/dev/null || true)
  staged_files=$(git diff --cached --name-only 2>/dev/null || true)
  
  # Combine and deduplicate
  modified_files=$(printf "%s\n%s" "$unstaged_files" "$staged_files" | sort -u | grep -v '^$' || true)
  
  if [[ -z "$modified_files" ]]; then
    log_verbose "No hay archivos modificados"
    return 1
  fi
  
  if [[ "$VERBOSE" == "true" ]]; then
    log_verbose "Archivos modificados encontrados (staged y unstaged):"
    while IFS= read -r file; do
      [[ -n "$file" ]] && log_verbose "  - $file"
    done <<< "$modified_files"
  fi
  
  local changelogs
  changelogs=$(echo "$modified_files" | grep -E '^.+/CHANGELOG\.md$' || true)
  
  if [[ -z "$changelogs" ]]; then
    log_verbose "No hay CHANGELOGs de módulos modificados"
    log_verbose "Solo se procesan CHANGELOGs en subdirectorios"
    return 1
  fi
  
  if [[ "$VERBOSE" == "true" ]]; then
    log_verbose "CHANGELOGs de módulos detectados:"
    while IFS= read -r file; do
      [[ -n "$file" ]] && log_verbose "  - $file"
    done <<< "$changelogs"
  fi
  
  echo "$changelogs"
  return 0
}

extract_module_from_changelog() {
  local changelog_path="$1"
  local module
  module=$(dirname "$changelog_path")
  echo "$module"
}

extract_version_from_changelog() {
  local changelog_path="$1"
  
  log_verbose "Extrayendo versión de $changelog_path"
  
  if [[ ! -f "$changelog_path" ]]; then
    log_error "Archivo no encontrado: $changelog_path"
    return 1
  fi
  
  # Extract the first version after [Unreleased]
  # Compatible with BSD awk (macOS) and GNU awk
  local version
  version=$(awk '
    /^## \[Unreleased\]/ { found_unreleased=1; next }
    found_unreleased && /^## \[?[0-9]/ {
      if (match($0, /[0-9]+\.[0-9]+\.[0-9]+/)) {
        print substr($0, RSTART, RLENGTH)
        exit
      }
    }
  ' "$changelog_path")
  
  if [[ -z "$version" ]]; then
    log_error "No se pudo extraer versión de $changelog_path"
    log_error "Asegúrate de que hay una sección [Unreleased] seguida de una versión [X.Y.Z]"
    return 1
  fi
  
  log_verbose "Versión encontrada: $version"
  echo "$version"
  return 0
}

# ============================================================================
# MODULE VALIDATION
# ============================================================================

validate_module() {
  local module="$1"
  local version="$2"
  local changelog="$3"
  
  log_verbose "Validando módulo '$module' versión '$version'..."
  
  # Check module directory exists
  if [[ ! -d "$ROOT_DIR/$module" ]]; then
    log_error "Directorio del módulo no existe: $module"
    return 1
  fi
  
  # Check CHANGELOG exists
  if [[ ! -f "$ROOT_DIR/$changelog" ]]; then
    log_error "CHANGELOG no existe: $changelog"
    return 1
  fi
  
  # Validate version format
  if ! validate_version_format "$version"; then
    log_error "Formato de versión inválido: $version (debe ser X.Y.Z)"
    return 1
  fi
  
  # Check if tag already exists
  local tag="$module/v$version"
  if check_tag_exists "$tag"; then
    log_error "El tag '$tag' ya existe"
    return 1
  fi
  
  log_verbose "✓ Módulo '$module' validado"
  return 0
}

run_release_check() {
  local module="$1"
  
  log_verbose "Ejecutando release-check para '$module'..."
  
  if [[ ! -f "$ROOT_DIR/$module/Makefile" ]]; then
    log_warning "No hay Makefile en $module, saltando release-check"
    return 0
  fi
  
  if ! grep -q "release-check:" "$ROOT_DIR/$module/Makefile"; then
    log_warning "No hay target release-check en $module/Makefile, saltando"
    return 0
  fi
  
  if [[ "$DRY_RUN" == "true" ]]; then
    log_warning "[DRY-RUN] Se ejecutaría: make -C $module release-check"
    return 0
  fi
  
  if ! make -C "$ROOT_DIR/$module" release-check > /dev/null 2>&1; then
    log_error "release-check falló para '$module'"
    return 1
  fi
  
  log_verbose "✓ release-check pasó para '$module'"
  return 0
}

# ============================================================================
# MODULE PROCESSING
# ============================================================================

process_module() {
  local changelog="$1"
  local module
  local version
  local tag
  
  module=$(extract_module_from_changelog "$changelog")
  
  if ! version=$(extract_version_from_changelog "$ROOT_DIR/$changelog"); then
    return 1
  fi
  
  if ! validate_module "$module" "$version" "$changelog"; then
    return 1
  fi
  
  if ! run_release_check "$module"; then
    return 1
  fi
  
  tag="$module/v$version"
  
  # Store module info
  MODULE_VERSIONS["$module"]="$version"
  MODULE_TAGS["$module"]="$tag"
  MODULE_CHANGELOGS["$module"]="$changelog"
  
  return 0
}

# ============================================================================
# GIT OPERATIONS
# ============================================================================

create_commit_and_tag() {
  local module="$1"
  local version="${MODULE_VERSIONS[$module]}"
  local tag="${MODULE_TAGS[$module]}"
  local changelog="${MODULE_CHANGELOGS[$module]}"
  
  log_info "Procesando $module v$version..."
  
  # Stage CHANGELOG
  run_cmd "git add '$ROOT_DIR/$changelog'"
  
  # Create commit
  local commit_msg="chore($module): release v$version

Update CHANGELOG.md for $module module release v$version"
  
  run_cmd "git commit -m '$commit_msg'"
  
  # Create tag
  run_cmd "git tag '$tag'"
  
  log_success "✓ Commit y tag creados para $module"
}

push_releases() {
  local tags=("${!MODULE_TAGS[@]}")
  local tag_list=""
  
  for module in "${tags[@]}"; do
    tag_list="$tag_list ${MODULE_TAGS[$module]}"
  done
  
  log_info "Haciendo push de commits..."
  run_cmd "git push '$REMOTE' HEAD"
  
  log_info "Haciendo push de tags..."
  run_cmd "git push '$REMOTE' $tag_list"
  
  log_success "✓ Push completado"
}

# ============================================================================
# DISPLAY FUNCTIONS
# ============================================================================

display_module_summary() {
  local module="$1"
  local version="${MODULE_VERSIONS[$module]}"
  local tag="${MODULE_TAGS[$module]}"
  
  echo -e "  ${GREEN}✓${NC} Versión: ${BOLD}$version${NC}"
  echo -e "  ${GREEN}✓${NC} Tag: ${BOLD}$tag${NC}"
  echo -e "  ${GREEN}✓${NC} Validaciones: ${GREEN}PASSED${NC}"
}

display_global_summary() {
  log_section "RESUMEN DE RELEASES"
  
  echo "Se crearán los siguientes releases:"
  echo ""
  
  local idx=1
  for module in "${MODULES_TO_PROCESS[@]}"; do
    local tag="${MODULE_TAGS[$module]}"
    echo -e "  ${BOLD}$idx.${NC} $tag"
    ((idx++))
  done
  
  echo ""
}

display_success_message() {
  echo ""
  log_success "Releases completados exitosamente!"
  echo ""
  echo "Los siguientes workflows de GitHub Actions se activarán:"
  
  for module in "${MODULES_TO_PROCESS[@]}"; do
    local tag="${MODULE_TAGS[$module]}"
    echo -e "  ${MAGENTA}•${NC} $tag → https://github.com/EduGoGroup/wapp-shared/actions"
  done
  
  echo ""
}

# ============================================================================
# MAIN WORKFLOW
# ============================================================================

select_modules() {
  local changelogs="$1"
  local changelog_array
  
  # Filter empty lines when creating array
  mapfile -t changelog_array < <(echo "$changelogs" | grep -v '^$')
  
  local num_changelogs=${#changelog_array[@]}
  
  log_info "Detectados $num_changelogs módulo(s) con cambios en CHANGELOG"
  echo "" >&2
  
  # If specific modules were specified, filter
  if [[ ${#SPECIFIED_MODULES[@]} -gt 0 ]]; then
    log_verbose "Filtrando por módulos especificados: ${SPECIFIED_MODULES[*]}"
    local filtered=()
    for changelog in "${changelog_array[@]}"; do
      local module
      module=$(extract_module_from_changelog "$changelog")
      for specified in "${SPECIFIED_MODULES[@]}"; do
        if [[ "$module" == "$specified" ]]; then
          filtered+=("$changelog")
          break
        fi
      done
    done
    
    if [[ ${#filtered[@]} -eq 0 ]]; then
      fail "Ninguno de los módulos especificados tiene cambios en CHANGELOG"
    fi
    
    changelog_array=("${filtered[@]}")
    num_changelogs=${#changelog_array[@]}
    log_info "Filtrando a $num_changelogs módulo(s)"
    echo "" >&2
  fi
  
  # Process all if --all or only one module
  if [[ "$PROCESS_ALL" == "true" ]] || [[ $num_changelogs -eq 1 ]]; then
    for changelog in "${changelog_array[@]}"; do
      echo "$changelog"
    done
    return 0
  fi
  
  # Interactive selection
  echo "Módulos disponibles:" >&2
  local idx=1
  for changelog in "${changelog_array[@]}"; do
    local module
    module=$(extract_module_from_changelog "$changelog")
    echo "  $idx. $module" >&2
    ((idx++))
  done
  echo "" >&2
  
  if ! confirm "¿Procesar todos los módulos?" "n"; then
    echo "" >&2
    echo "Especifica los números de los módulos a procesar (separados por espacio):" >&2
    read -r -p "> " selection
    
    local selected=()
    for num in $selection; do
      if [[ "$num" =~ ^[0-9]+$ ]] && [[ $num -ge 1 ]] && [[ $num -le $num_changelogs ]]; then
        selected+=("${changelog_array[$((num-1))]}")
      fi
    done
    
    if [[ ${#selected[@]} -eq 0 ]]; then
      fail "No se seleccionó ningún módulo válido"
    fi
    
    for changelog in "${selected[@]}"; do
      echo "$changelog"
    done
  else
    for changelog in "${changelog_array[@]}"; do
      echo "$changelog"
    done
  fi
}

main() {
  cd "$ROOT_DIR"
  
  log_verbose "Iniciando auto-release script..."
  log_verbose "Directorio raíz: $ROOT_DIR"
  
  # Validations
  check_git_repo
  check_remote
  
  # Detect modified CHANGELOGs
  local changelogs
  if ! changelogs=$(detect_modified_changelogs); then
    fail "No hay CHANGELOGs modificados. Modifica un CHANGELOG.md antes de ejecutar este script."
  fi
  
  # Select modules to process
  local selected_changelogs
  selected_changelogs=$(select_modules "$changelogs")
  
  # Process each module
  local idx=1
  local total
  total=$(echo "$selected_changelogs" | wc -l | tr -d ' ')
  
  while IFS= read -r changelog; do
    [[ -z "$changelog" ]] && continue
    
    local module
    module=$(extract_module_from_changelog "$changelog")
    
    log_section "Módulo $idx/$total: $module"
    
    if process_module "$changelog"; then
      MODULES_TO_PROCESS+=("$module")
      display_module_summary "$module"
    else
      log_error "Falló el procesamiento de '$module'"
      if [[ "$PROCESS_ALL" == "true" ]]; then
        log_warning "Continuando con el siguiente módulo..."
        ((idx++))
        continue
      else
        fail "Abortando debido a error en '$module'"
      fi
    fi
    
    ((idx++))
  done <<< "$selected_changelogs"
  
  # Check if any modules were successfully processed
  if [[ ${#MODULES_TO_PROCESS[@]} -eq 0 ]]; then
    fail "No se pudo procesar ningún módulo"
  fi
  
  # Display summary and confirm
  display_global_summary
  
  if ! confirm "¿Continuar con los commits, tags y push?" "n"; then
    log_warning "Operación cancelada por el usuario"
    exit 0
  fi
  
  echo ""
  
  # Create commits and tags
  for module in "${MODULES_TO_PROCESS[@]}"; do
    create_commit_and_tag "$module"
  done
  
  # Push everything
  push_releases
  
  # Success message
  display_success_message
}

# ============================================================================
# USAGE AND ARGUMENT PARSING
# ============================================================================

usage() {
  cat <<EOF
${BOLD}Uso:${NC} $SCRIPT_NAME [OPTIONS] [MODULES...]

${BOLD}Descripción:${NC}
  Detecta CHANGELOGs modificados, extrae versiones, crea commits y tags,
  y los pushea para activar el workflow de GitHub Actions release.

${BOLD}Opciones:${NC}
  -h, --help          Mostrar esta ayuda
  -d, --dry-run       Simular sin hacer cambios reales
  -v, --verbose       Modo verbose para debugging
  -a, --all           Procesar todos los módulos modificados
  -r, --remote NAME   Especificar remote (default: origin)
  -y, --yes           Auto-confirmar (no interactivo)

${BOLD}Argumentos:${NC}
  MODULES             Módulos específicos a procesar (ej: postgres mongodb)

${BOLD}Ejemplos:${NC}
  $SCRIPT_NAME                        # Modo interactivo
  $SCRIPT_NAME --all                  # Todos los módulos
  $SCRIPT_NAME postgres               # Solo postgres
  $SCRIPT_NAME postgres mongodb       # Múltiples específicos
  $SCRIPT_NAME --dry-run -v           # Dry-run con verbose
  $SCRIPT_NAME --all --yes            # Todos sin confirmación

${BOLD}Flujo:${NC}
  1. Detecta CHANGELOGs modificados (sin commitear)
  2. Extrae la versión más reciente de cada CHANGELOG
  3. Valida módulos y versiones
  4. Ejecuta release-check para cada módulo
  5. Crea commits y tags
  6. Pushea al remote para activar GitHub Actions

${BOLD}Requisitos:${NC}
  - Estar en un repositorio Git
  - Tener cambios sin commitear en archivos CHANGELOG.md
  - Los CHANGELOGs deben tener formato válido con sección [Unreleased]
  - Los módulos deben tener Makefile con target release-check (opcional)

EOF
}

# Parse arguments
while [[ $# -gt 0 ]]; do
  case "$1" in
    -h|--help)
      usage
      exit 0
      ;;
    -d|--dry-run)
      DRY_RUN=true
      shift
      ;;
    -v|--verbose)
      VERBOSE=true
      shift
      ;;
    -a|--all)
      PROCESS_ALL=true
      shift
      ;;
    -r|--remote)
      REMOTE="$2"
      shift 2
      ;;
    -y|--yes)
      AUTO_YES=true
      shift
      ;;
    -*)
      log_error "Opción desconocida: $1"
      echo ""
      usage
      exit 1
      ;;
    *)
      SPECIFIED_MODULES+=("$1")
      shift
      ;;
  esac
done

# Run main
main

# Made with Bob
