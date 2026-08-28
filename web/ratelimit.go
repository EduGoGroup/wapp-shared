package web

import (
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// Valores por defecto del limitador. Son configurables a propósito: un test
// determinista necesita poner el TTL en milisegundos, y con constantes de
// paquete la purga solo se podía probar con sleeps largos o no probarse.
const (
	DefaultRateLimitTTL        = 10 * time.Minute
	DefaultRateLimitPurgeEvery = time.Minute
)

// RateLimiterOptions configura el limitador por clave.
type RateLimiterOptions struct {
	// RPS es la recarga del token bucket (peticiones por segundo).
	RPS float64
	// Burst es la ráfaga inmediata admitida.
	Burst int
	// TTL es la inactividad tras la cual se desaloja una clave; <= 0 cae a
	// DefaultRateLimitTTL.
	TTL time.Duration
	// PurgeEvery es cada cuánto, COMO MUCHO, intenta barrer Allow; <= 0 cae a
	// DefaultRateLimitPurgeEvery.
	PurgeEvery time.Duration
}

// KeyedRateLimiter es un rate-limit en memoria por clave (IP del cliente o, si
// hay sesión, user_id) sobre el token bucket de golang.org/x/time/rate. Cada
// clave tiene su propio bucket.
//
// # La purga es PEREZOSA, y no hay goroutine de barrido
//
// El desalojo de claves inactivas lo hace Allow de forma amortizada (como mucho
// una vez cada PurgeEvery), no una goroutine de fondo. Es deliberado: quien
// construye el limitador suele ser un NewRouter que no expone ciclo de vida al
// llamador, así que una goroutine de barrido solo tenía dos finales posibles
// —quedarse viva para siempre, o pararse y dejar el mapa creciendo sin tope, una
// entrada por IP—. Las dos implementaciones que este módulo reconcilia cayeron
// cada una en uno de esos dos finales. Sin goroutine no hay ninguno de los dos.
//
// CONTRAPARTIDA ASUMIDA: si el tráfico cesa por completo, el mapa se queda con
// las entradas que hubiera —no crece más, pero tampoco se vacía— hasta la
// siguiente petición, que las barre.
//
// NOTA (INV-6): el estado es por instancia, sin broker. Con varias réplicas
// detrás de un balanceador el límite efectivo se multiplica por el número de
// instancias; un límite global exigiría un backend compartido y NO se implementa.
type KeyedRateLimiter struct {
	mu         sync.Mutex
	buckets    map[string]*bucketEntry
	rps        rate.Limit
	burst      int
	ttl        time.Duration // inactividad tras la cual se desaloja una clave.
	purgeEvery time.Duration // cada cuánto, como mucho, Allow intenta el barrido.
	lastPurge  time.Time
	// now es el reloj. Es un campo y no una variable de paquete a propósito: los
	// tests corren en paralelo y con -race, y un global mutable sería una carrera.
	now       func() time.Time
	closeOnce sync.Once
}

// bucketEntry es el bucket de una clave junto a la última vez que se usó.
type bucketEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// NewKeyedRateLimiter crea el limitador. No arranca ninguna goroutine: el
// barrido de claves inactivas ocurre dentro de Allow (ver KeyedRateLimiter).
func NewKeyedRateLimiter(opts RateLimiterOptions) *KeyedRateLimiter {
	if opts.TTL <= 0 {
		opts.TTL = DefaultRateLimitTTL
	}
	if opts.PurgeEvery <= 0 {
		opts.PurgeEvery = DefaultRateLimitPurgeEvery
	}
	now := time.Now
	return &KeyedRateLimiter{
		buckets:    make(map[string]*bucketEntry),
		rps:        rate.Limit(opts.RPS),
		burst:      opts.Burst,
		ttl:        opts.TTL,
		purgeEvery: opts.PurgeEvery,
		lastPurge:  now(),
		now:        now,
	}
}

// Allow consume un token del bucket de la clave (creándolo si no existía) y de
// paso barre las claves inactivas. Devuelve false si la clave agotó su ráfaga.
func (k *KeyedRateLimiter) Allow(key string) bool {
	k.mu.Lock()
	now := k.now()
	k.purgeLocked(now)

	entry, ok := k.buckets[key]
	if !ok {
		entry = &bucketEntry{limiter: rate.NewLimiter(k.rps, k.burst)}
		k.buckets[key] = entry
	}
	entry.lastSeen = now
	k.mu.Unlock()

	// Fuera del mutex: *rate.Limiter ya es seguro para uso concurrente y así el
	// candado del mapa no se sostiene durante la contabilidad del bucket.
	return entry.limiter.Allow()
}

// purgeLocked desaloja las claves inactivas más viejas que el TTL. Amortizado:
// no hace nada si no ha pasado PurgeEvery desde el último barrido, de modo que
// el coste por petición es una comparación de tiempos. Exige k.mu tomado.
func (k *KeyedRateLimiter) purgeLocked(now time.Time) {
	if now.Sub(k.lastPurge) < k.purgeEvery {
		return
	}
	k.lastPurge = now
	for key, entry := range k.buckets {
		if now.Sub(entry.lastSeen) > k.ttl {
			delete(k.buckets, key)
		}
	}
}

// Close libera de golpe las entradas del limitador. Es la función de limpieza
// que el dueño del ciclo de vida (el bootstrap, en un defer al apagar) puede
// entregar como cleanup del router.
//
// Es IDEMPOTENTE (sync.Once): nada impide que un caller la invoque en un defer y
// también en un camino de error, y ese doble cierre no puede tener efecto la
// segunda vez.
//
// Y NO inhabilita el limitador: Allow sigue atendiendo y purgando después de
// Close, porque hay callers que cierran y luego siguen sirviendo peticiones con
// el mismo router. Cerrar es soltar memoria, no apagar la defensa.
func (k *KeyedRateLimiter) Close() {
	k.closeOnce.Do(func() {
		k.mu.Lock()
		defer k.mu.Unlock()
		k.buckets = make(map[string]*bucketEntry)
		k.lastPurge = k.now()
	})
}

// Len es el número de claves vivas. Sirve para observabilidad (una gauge).
//
// 🔴 NO es la forma de probar que la purga funciona: mirar el mapa desde fuera
// prueba el mapa, no el desalojo. Que una clave se desalojó se observa por
// COMPORTAMIENTO — con la recarga a ~0, la única forma de que una clave ya
// cortada vuelva a pasar es que su entrada se fuera y naciera un bucket nuevo.
func (k *KeyedRateLimiter) Len() int {
	k.mu.Lock()
	defer k.mu.Unlock()
	return len(k.buckets)
}

// Prefijos del espacio de claves del limitador. Sin ellos, un user_id que
// coincidiera con una IP compartiría bucket con ella.
const (
	rateKeyUserPrefix = "u:"
	rateKeyIPPrefix   = "ip:"
)

// UserRateKey es la clave de limitación de un usuario identificado.
func UserRateKey(userID string) string { return rateKeyUserPrefix + userID }

// IPRateKey es la clave de limitación de una IP.
func IPRateKey(ip string) string { return rateKeyIPPrefix + ip }

// RateKey elige la clave: el user_id si hay sesión (la más específica gana), si
// no la IP del cliente. Los prefijos evitan la colisión entre los dos espacios.
func RateKey(userID, clientIP string) string {
	if userID != "" {
		return UserRateKey(userID)
	}
	return IPRateKey(clientIP)
}
