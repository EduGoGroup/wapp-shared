// Package jwt emite y valida los tokens del ecosistema wApp: JWT de usuario
// (ES256 asimétrico — ADR-0019 — o HS256 simétrico), service tokens M2M por
// scopes y refresh tokens opacos.
//
// Es lógica PURA: no toca base de datos ni HTTP. Los consumidores (p. ej.
// cloud-platform) construyen sus casos de uso IAM encima de estas piezas.
//
// El modelo de claims está SIMPLIFICADO a {tenant_id, user_id, roles, grants}:
// es una copia-adaptación del módulo auth de EduGo SIN la jerarquía
// escuela/unidad/ward (ADR-0004 copia-adaptación; cero import a repos EduGo).
//
// La firma ES256 estampa un `kid` en el header ([JWTManager.WithKid]) y
// [MultiVerifier] selecciona la llave por ese `kid`, lo que permite rotar
// algoritmos y llaves sin invalidar los tokens en vuelo.
//
// Los grants viajan en los claims como [Grants], alias de rbac.Grants: este
// paquete los transporta, el paquete rbac los evalúa.
package jwt
