// Package config carga configuracion para el ecosistema wApp combinando un
// archivo YAML opcional con un overlay de variables de entorno.
//
// El punto de entrada es [Loader], construido con [New] y opciones funcionales:
//
//	loader := config.New(
//		config.WithFile("config.yaml"),
//		config.WithEnvPrefix("WAPP_"),
//	)
//
//	var cfg struct {
//		Puerto int    `yaml:"puerto"`
//		Host   string `yaml:"host"`
//	}
//	if err := loader.Unmarshal(&cfg); err != nil {
//		// manejar error
//	}
//
//	nivel := loader.GetString("LOG_LEVEL", "info") // lee WAPP_LOG_LEVEL
//
// [Loader.Unmarshal] no falla si el archivo configurado no existe: simplemente
// no aplica overlay desde archivo. Los getters tipados leen de variables de
// entorno usando el prefijo configurado y devuelven el default cuando la
// variable no esta definida o no es parseable.
package config
