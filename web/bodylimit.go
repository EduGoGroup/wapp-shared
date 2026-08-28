package web

// BodyLimit decide a qué peticiones se les acota el cuerpo.
//
// Se aplica por LISTA DE RUTAS y no globalmente a propósito: hay pantallas que
// publican documentos legítimamente grandes, y meterles un techo por la puerta
// de atrás sería cambiar el comportamiento de una pantalla ajena sin que nadie
// lo pidiera.
//
// 🔴 El middleware que use esto tiene que montarse ANTES del de CSRF: el CSRF lee
// el formulario para comparar el token y con eso consume el cuerpo entero, así
// que un tope montado después llega cuando el daño ya está hecho.
type BodyLimit struct {
	limit   int64
	guarded map[string]bool
}

// NewBodyLimit construye el tope para las rutas indicadas (rutas exactas).
func NewBodyLimit(limit int64, paths ...string) *BodyLimit {
	guarded := make(map[string]bool, len(paths))
	for _, p := range paths {
		guarded[p] = true
	}
	return &BodyLimit{limit: limit, guarded: guarded}
}

// Limit es el techo en bytes.
func (b *BodyLimit) Limit() int64 { return b.limit }

// Guards dice si a esta petición se le aplica el tope: solo a los métodos que
// mutan estado y solo en las rutas declaradas.
func (b *BodyLimit) Guards(method, path string) bool {
	return IsUnsafeMethod(method) && b.guarded[path]
}
