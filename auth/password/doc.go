// Package password hashea y verifica contraseñas con bcrypt (cost 12).
//
// Ofrece dos superficies equivalentes: las funciones [HashPassword] y
// [VerifyPassword] para el uso directo, y la interfaz [Hasher] con su
// implementación [DefaultHasher] para los consumidores que inyectan la
// dependencia en sus casos de uso.
//
// Es lógica PURA: no toca base de datos ni HTTP.
package password
