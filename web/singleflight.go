package web

import (
	"errors"
	"fmt"
	"sync"
)

// ErrRefreshPanic envuelve el pánico de un fn() para los que esperaban en la
// misma clave: sin él recibirían el valor CERO y un error nil, es decir, un
// refresh fallido disfrazado de éxito.
var ErrRefreshPanic = errors.New("la operación de refresh entró en pánico")

// RefreshGroup es un single-flight por clave: N peticiones concurrentes que
// quieren refrescar el MISMO token disparan una sola llamada al upstream y todas
// reciben su resultado.
//
// Es genérico porque las dos consolas que lo tenían duplicado lo escribieron con
// tipos distintos —una con su propio *AuthResult, la otra con `any`— y ninguna
// de las dos formas servía para las dos: la concreta no se puede compartir y la
// de `any` obliga a un type assertion en cada llamador.
//
// El cero de RefreshGroup es utilizable.
type RefreshGroup[T any] struct {
	mu    sync.Mutex
	calls map[string]*refreshCall[T]
}

// refreshCall es una ejecución en curso: los que esperan bloquean en done y
// luego leen val/err.
type refreshCall[T any] struct {
	done chan struct{}
	val  T
	err  error
}

// NewRefreshGroup crea un grupo vacío. El cero del tipo también vale; esto
// existe por simetría con el resto del paquete.
func NewRefreshGroup[T any]() *RefreshGroup[T] {
	return &RefreshGroup[T]{calls: make(map[string]*refreshCall[T])}
}

// Do ejecuta fn una sola vez por key; los llamadores concurrentes con la misma
// key esperan y reciben el mismo resultado.
//
// La limpieza va en un DEFER, y esto no es estilo: si fn() entra en pánico y la
// limpieza estuviera al final del cuerpo, el canal nunca se cerraría y el mapa
// nunca soltaría la clave, así que TODA petición posterior con esa misma key se
// quedaría colgada para siempre, sin timeout, ocupando una goroutine. El pánico
// se vuelve a lanzar tras limpiar, para que el recuperador de arriba (p. ej.
// gin.Recovery) lo siga viendo; a los que esperaban se les entrega
// ErrRefreshPanic en vez de un falso éxito.
func (g *RefreshGroup[T]) Do(key string, fn func() (T, error)) (T, error) {
	g.mu.Lock()
	if g.calls == nil {
		g.calls = make(map[string]*refreshCall[T])
	}
	if call, ok := g.calls[key]; ok {
		g.mu.Unlock()
		<-call.done
		return call.val, call.err
	}
	call := &refreshCall[T]{done: make(chan struct{})}
	g.calls[key] = call
	g.mu.Unlock()

	defer func() {
		r := recover()
		if r != nil {
			call.err = fmt.Errorf("%w: %v", ErrRefreshPanic, r)
		}
		g.mu.Lock()
		delete(g.calls, key)
		g.mu.Unlock()
		close(call.done)
		if r != nil {
			panic(r)
		}
	}()

	call.val, call.err = fn()
	return call.val, call.err
}

// InFlight es el número de ejecuciones en curso. Sirve para verificar que una
// clave se libera pase lo que pase.
func (g *RefreshGroup[T]) InFlight() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return len(g.calls)
}
