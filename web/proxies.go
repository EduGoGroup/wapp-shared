package web

import "strings"

// ParseTrustedProxies convierte el CSV de proxies de confianza (IPs o CIDRs) a
// lista, descartando vacíos. Devuelve nil cuando no hay ninguno.
//
// La lista vacía NO es un descuido: es la postura por defecto. Sin proxies de
// confianza, la IP del cliente se resuelve desde la conexión y se IGNORA el
// X-Forwarded-For, que es lo que blinda el rate-limit por IP del login —la única
// defensa anti fuerza-bruta— contra la suplantación de esa cabecera. Solo se
// confía en la lista explícita cuando de verdad hay un proxy delante.
func ParseTrustedProxies(csv string) []string {
	var proxies []string
	for _, raw := range strings.Split(csv, ",") {
		p := strings.TrimSpace(raw)
		if p == "" {
			continue
		}
		proxies = append(proxies, p)
	}
	return proxies
}
