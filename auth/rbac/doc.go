// Package rbac evalúa los permisos del ecosistema wApp: el matcher glob de
// grants y la resolución de la cadena de roles.
//
// Los grants siguen la gramática glob `recurso.accion` con comodines (`*`,
// `prefix.*`, `*.suffix`, `prefix.*.suffix`) y precedencia deny-sobre-allow,
// con default DENY: [PermissionMatches] decide si un pattern cubre un permiso
// y [EvaluateGrants] resuelve el conjunto completo.
//
// La herencia de roles se resuelve con [ResolveRoleChain] (guard de ciclos) y
// se aplana con [MergeGrantChain].
//
// Es lógica PURA: no toca base de datos ni HTTP.
package rbac
