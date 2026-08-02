package rbac

import "context"

// PermissionEvaluator define el contrato para la evaluación de permisos y políticas RBAC.
type PermissionEvaluator interface {
	CheckPermission(ctx context.Context, requiredPermission string) bool
}
