// Package auth ofrece las primitivas de autenticación y autorización del
// ecosistema wApp: emisión y validación de JWT de usuario (HS256 simétrico o,
// opcionalmente, ES256 asimétrico — ADR-0019), service tokens M2M, hashing de
// contraseñas (bcrypt), refresh tokens opacos y el matcher glob de permisos
// (RBAC) con resolución de cadena de roles.
//
// Es lógica PURA: no toca base de datos ni HTTP. Los consumidores (p.ej.
// cloud-platform) construyen sus casos de uso IAM encima de estas piezas.
//
// El modelo de claims está SIMPLIFICADO a {tenant_id, user_id, roles, grants}:
// es una copia-adaptación del módulo auth de EduGo SIN la jerarquía
// escuela/unidad/ward (ADR-0004 copia-adaptación; cero import a repos EduGo).
//
// Los grants siguen la gramática glob `recurso.accion` con comodines (`*`,
// `prefix.*`, `*.suffix`) y precedencia deny-sobre-allow, con default DENY.
package auth
