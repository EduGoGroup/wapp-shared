package web

import (
	"sync"
	"testing"
	"time"
)

// relojFalso permite mover el tiempo a mano: la purga se verifica sin dormir ni
// un milisegundo real, así que el test no puede volverse flaky por carga.
type relojFalso struct {
	mu    sync.Mutex
	ahora time.Time
}

func (r *relojFalso) now() time.Time {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.ahora
}

func (r *relojFalso) avanza(d time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.ahora = r.ahora.Add(d)
}

// conReloj construye un limitador con reloj controlado. El constructor sella
// lastPurge con el reloj REAL, así que se realinea con el falso: es lo que el
// propio constructor habría hecho de haberlo tenido.
func conReloj(opts RateLimiterOptions) (*KeyedRateLimiter, *relojFalso) {
	reloj := &relojFalso{ahora: time.Date(2026, 8, 28, 12, 0, 0, 0, time.UTC)}
	k := NewKeyedRateLimiter(opts)
	k.now = reloj.now
	k.lastPurge = reloj.now()
	return k, reloj
}

// TestKeyedRateLimiter_ElDesalojoSeObservaPorComportamiento es LA prueba de que
// la purga perezosa funciona, y no mira el mapa a propósito: espiarlo probaría
// el mapa, no el desalojo.
//
// Con la recarga a ~0 (1 token cada 1000 s) el bucket NO se rellena durante el
// test, así que una clave que ya se cortó no puede volver a pasar… salvo que su
// entrada se haya desalojado y haya nacido un bucket nuevo con la ráfaga entera.
// Ese 200 final es la única señal posible del desalojo.
func TestKeyedRateLimiter_ElDesalojoSeObservaPorComportamiento(t *testing.T) {
	t.Parallel()

	k, reloj := conReloj(RateLimiterOptions{
		RPS: 0.001, Burst: 1,
		TTL: 30 * time.Millisecond, PurgeEvery: time.Millisecond,
	})
	clave := IPRateKey("203.0.113.7")

	if !k.Allow(clave) {
		t.Fatal("la 1ª petición (dentro de la ráfaga) debía pasar")
	}
	if k.Allow(clave) {
		t.Fatal("la 2ª (ráfaga agotada, sin recarga observable) debía cortarse")
	}

	// El reloj supera el TTL: la entrada queda inactiva y la siguiente petición
	// tiene que barrerla antes de atenderse.
	reloj.avanza(100 * time.Millisecond)

	if !k.Allow(clave) {
		t.Fatal("tras superar el TTL la petición debía pasar: la entrada NO se purgó " +
			"(con rps ≈ 0 el bucket viejo no puede haberse recargado)")
	}
}

// TestKeyedRateLimiter_PurgaPerezosaAmortizada prueba el MECANISMO: el barrido
// ocurre dentro de Allow, como mucho una vez por PurgeEvery, sin depender de
// ninguna goroutine.
func TestKeyedRateLimiter_PurgaPerezosaAmortizada(t *testing.T) {
	t.Parallel()

	k, reloj := conReloj(RateLimiterOptions{
		RPS: 1000, Burst: 1000,
		TTL: 50 * time.Millisecond, PurgeEvery: 10 * time.Millisecond,
	})

	for _, ip := range []string{"10.0.0.1", "10.0.0.2", "10.0.0.3"} {
		k.Allow(IPRateKey(ip))
	}
	if got := k.Len(); got != 3 {
		t.Fatalf("entradas tras 3 claves distintas = %d, want 3", got)
	}

	// Antes del intervalo de barrido no se barre nada: el coste por petición es
	// una comparación de tiempos, no un recorrido del mapa.
	reloj.avanza(5 * time.Millisecond)
	k.Allow(IPRateKey("10.0.0.4"))
	if got := k.Len(); got != 4 {
		t.Fatalf("no debía barrerse aún (amortizado); entradas = %d, want 4", got)
	}

	// Pasados el TTL y el intervalo, las inactivas caen y la recién vista no.
	reloj.avanza(100 * time.Millisecond)
	k.Allow(IPRateKey("10.0.0.5"))
	if got := k.Len(); got != 1 {
		t.Fatalf("entradas tras superar el TTL = %d, want 1 (solo la recién vista)", got)
	}
}

// TestKeyedRateLimiter_CloseEsIdempotenteYNoInhabilita.
//
// No basta con «el segundo Close no entra en pánico»: eso sería VACUO en cuanto
// no hay ningún canal que cerrar dos veces. Lo que se comprueba es que el
// segundo cierre es un NO-OP —las entradas creadas tras el primero sobreviven—,
// y eso sí cae si alguien quita el sync.Once. Y que tras cerrar el limitador
// SIGUE atendiendo: hay callers que cierran y después sirven peticiones con el
// mismo router.
func TestKeyedRateLimiter_CloseEsIdempotenteYNoInhabilita(t *testing.T) {
	t.Parallel()

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("el segundo Close() entró en pánico: %v", r)
		}
	}()

	k := NewKeyedRateLimiter(RateLimiterOptions{RPS: 5, Burst: 10})

	k.Allow(IPRateKey("10.0.0.1"))
	k.Close()
	if got := k.Len(); got != 0 {
		t.Fatalf("entradas tras Close() = %d, want 0 (Close libera el mapa)", got)
	}

	if !k.Allow(IPRateKey("10.0.0.2")) {
		t.Fatal("tras Close() el limitador debe seguir atendiendo")
	}

	k.Close() // segunda llamada: ni pánico ni efecto.
	if got := k.Len(); got != 1 {
		t.Fatalf("entradas tras el 2º Close() = %d, want 1: Close no es idempotente, "+
			"volvió a vaciar el mapa", got)
	}
}

// TestKeyedRateLimiter_LaRafagaEsPorClave: agotar una clave no afecta a otra, y
// los prefijos mantienen separados los dos espacios de nombres.
func TestKeyedRateLimiter_LaRafagaEsPorClave(t *testing.T) {
	t.Parallel()

	k, _ := conReloj(RateLimiterOptions{RPS: 0.001, Burst: 1})

	if !k.Allow(IPRateKey("203.0.113.7")) {
		t.Fatal("la 1ª petición debía pasar (dentro de la ráfaga)")
	}
	if k.Allow(IPRateKey("203.0.113.7")) {
		t.Fatal("la 2ª petición de la misma clave debía cortarse")
	}
	if !k.Allow(IPRateKey("203.0.113.8")) {
		t.Fatal("otra IP tiene su propio bucket y debía pasar")
	}
	// Un user_id que coincida con una IP NO puede compartir bucket con ella.
	if !k.Allow(UserRateKey("203.0.113.7")) {
		t.Fatal("el espacio de claves de usuario es distinto del de IP")
	}
}

func TestRateKey_LaClaveMasEspecificaGana(t *testing.T) {
	t.Parallel()

	if got := RateKey("u-42", "203.0.113.7"); got != "u:u-42" {
		t.Errorf("con sesión debe ganar el user_id, got %q", got)
	}
	if got := RateKey("", "203.0.113.7"); got != "ip:203.0.113.7" {
		t.Errorf("sin sesión debe usarse la IP, got %q", got)
	}
}

// TestKeyedRateLimiter_Concurrente comprueba que ni el barrido ni la creación de
// buckets se pisan (corre con -race en el gate).
func TestKeyedRateLimiter_Concurrente(t *testing.T) {
	t.Parallel()

	k, reloj := conReloj(RateLimiterOptions{
		RPS: 1000, Burst: 1000,
		TTL: time.Millisecond, PurgeEvery: time.Millisecond,
	})

	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				k.Allow(IPRateKey(string(rune('a' + n%26))))
				reloj.avanza(time.Microsecond)
			}
		}(i)
	}
	wg.Wait()
	k.Close()
}
