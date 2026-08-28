package web

import (
	"errors"
	"sync"
	"testing"
	"time"
)

// TestRefreshGroup_SobreviveAUnPanicoEnFn es el test de regresión portado de la
// consola de plataforma: sin el defer de limpieza, un pánico dentro de fn()
// dejaba la clave en el mapa y el canal sin cerrar, así que TODA petición
// posterior con la misma clave se quedaba colgada para siempre, sin timeout,
// ocupando una goroutine.
func TestRefreshGroup_SobreviveAUnPanicoEnFn(t *testing.T) {
	t.Parallel()

	g := NewRefreshGroup[string]()

	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Error("el pánico debe seguir propagándose hacia arriba (gin.Recovery)")
			}
		}()
		val, err := g.Do("rt-panic", func() (string, error) {
			panic("fn entra en pánico a propósito")
		})
		t.Errorf("Do debía propagar el pánico y no devolvió: (%q, %v)", val, err)
	}()

	hecho := make(chan struct{})
	go func() {
		defer close(hecho)
		val, err := g.Do("rt-panic", func() (string, error) { return "ok-tras-el-panico", nil })
		if err != nil {
			t.Errorf("segunda llamada con la misma key: err = %v, want nil", err)
		}
		if val != "ok-tras-el-panico" {
			t.Errorf("segunda llamada con la misma key: val = %q", val)
		}
	}()

	select {
	case <-hecho:
	case <-time.After(2 * time.Second):
		t.Fatal("la segunda llamada con la misma key se quedó colgada: el pánico dejó la clave sin liberar")
	}

	if got := g.InFlight(); got != 0 {
		t.Errorf("la clave debía borrarse del mapa tras el pánico; InFlight = %d", got)
	}
}

// TestRefreshGroup_LosQueEsperanNoRecibenUnFalsoExito: si fn() entra en pánico,
// el que esperaba no puede recibir el valor CERO con error nil —eso sería un
// refresh fallido disfrazado de sesión renovada.
func TestRefreshGroup_LosQueEsperanNoRecibenUnFalsoExito(t *testing.T) {
	t.Parallel()

	g := NewRefreshGroup[string]()
	entro := make(chan struct{})
	suelta := make(chan struct{})

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer func() {
			if r := recover(); r == nil {
				t.Error("el pánico debía propagarse también al que lo provocó")
			}
		}()
		val, err := g.Do("k", func() (string, error) {
			close(entro)
			<-suelta
			panic("boom")
		})
		t.Errorf("Do debía propagar el pánico y no devolvió: (%q, %v)", val, err)
	}()

	<-entro
	var val string
	var err error
	wg.Add(1)
	go func() {
		defer wg.Done()
		val, err = g.Do("k", func() (string, error) { return "no-debería-ejecutarse", nil })
	}()
	// Le damos tiempo al segundo a entrar en la espera antes de soltar el pánico.
	time.Sleep(20 * time.Millisecond)
	close(suelta)
	wg.Wait()

	if !errors.Is(err, ErrRefreshPanic) {
		t.Fatalf("el que esperaba recibió err = %v, want ErrRefreshPanic", err)
	}
	if val != "" {
		t.Errorf("val = %q, want el cero del tipo", val)
	}
}

// TestRefreshGroup_UnaSolaEjecucionPorClave es la razón de existir del grupo.
func TestRefreshGroup_UnaSolaEjecucionPorClave(t *testing.T) {
	t.Parallel()

	g := NewRefreshGroup[int]()
	var veces int32
	var mu sync.Mutex
	arranca := make(chan struct{})
	suelta := make(chan struct{})

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-arranca
			v, err := g.Do("misma", func() (int, error) {
				mu.Lock()
				veces++
				mu.Unlock()
				<-suelta
				return 7, nil
			})
			if err != nil || v != 7 {
				t.Errorf("Do devolvió (%d, %v)", v, err)
			}
		}()
	}
	close(arranca)
	time.Sleep(30 * time.Millisecond)
	close(suelta)
	wg.Wait()

	mu.Lock()
	defer mu.Unlock()
	if veces != 1 {
		t.Fatalf("fn se ejecutó %d veces, want 1", veces)
	}
	if got := g.InFlight(); got != 0 {
		t.Errorf("InFlight = %d, want 0", got)
	}
}

// TestRefreshGroup_ElCeroDelTipoEsUsable.
func TestRefreshGroup_ElCeroDelTipoEsUsable(t *testing.T) {
	t.Parallel()

	var g RefreshGroup[string]
	v, err := g.Do("k", func() (string, error) { return "va", nil })
	if err != nil || v != "va" {
		t.Fatalf("Do sobre el cero del tipo devolvió (%q, %v)", v, err)
	}
}
