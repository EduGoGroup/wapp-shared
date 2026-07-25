package config

import "os"

// EnvProvider abstrae el acceso a variables de entorno para permitir
// pruebas unitarias y entornos aislados sin modificar el SO.
type EnvProvider interface {
	LookupEnv(key string) (string, bool)
}

// FileReader abstrae la lectura de archivos de configuración.
type FileReader interface {
	ReadFile(path string) ([]byte, error)
}

// osEnvProvider implementa EnvProvider utilizando os.LookupEnv.
type osEnvProvider struct{}

func (osEnvProvider) LookupEnv(key string) (string, bool) {
	return os.LookupEnv(key)
}

// osFileReader implementa FileReader utilizando os.ReadFile.
type osFileReader struct{}

func (osFileReader) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

// MapEnvProvider implementa EnvProvider sobre un mapa en memoria,
// ideal para inyección en pruebas unitarias.
type MapEnvProvider map[string]string

func (m MapEnvProvider) LookupEnv(key string) (string, bool) {
	v, ok := m[key]
	return v, ok
}
