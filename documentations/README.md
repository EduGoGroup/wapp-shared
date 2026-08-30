# `wapp-shared` — documentación de la pieza

**Qué es.** El monorepo Go **multi-módulo** de librerías compartidas del ecosistema **wApp**: once
módulos Go independientes (`auth`, `config`, `envelope`, `health`, `iam`, `intents`, `llm`,
`logger`, `textmatch`, `ui`, `web`), cada uno con su `go.mod`, su `CHANGELOG.md` y su propia línea
de versiones. **No produce ni un binario y no sirve ni una ruta.** Existe para que la política
transversal del ecosistema —logging, configuración, criptografía, JWT, middleware web, cliente de
identidad, puerto del LLM, tokens de diseño— tenga **un solo dueño** en vez de una copia por repo.

**Para qué existe.** Porque ya hubo copias, y divergieron. Los módulos `web` e `iam` nacieron
literalmente **reconciliando dos forks** que habían tomado decisiones distintas sobre la misma
cosa. Aquí se decide una vez, se publica con un tag `<modulo>/vX.Y.Z`, y los seis repos consumidores
pinean la versión que quieren.

**Contexto en el que se usa.** wApp es un ecosistema de mensajería sobre WhatsApp cuyo núcleo corre
24/7 en el equipo del cliente (el **Edge**), gobernado por una plataforma cloud modular. Este repo
alimenta a los dos lados a la vez, y por eso una decisión tomada aquí se hereda entera: hay
invariantes del ecosistema que se pueden violar desde una librería sin que nada compile en rojo.
**Están escritos en `constitucion.md` y hay que leerlos antes de tocar `envelope`, `auth` o `llm`.**

---

## Los documentos

| Documento | Qué encontrarás |
|---|---|
| [`constitucion.md`](constitucion.md) | 🔴 **Empieza aquí.** Los invariantes que no se pueden violar (los del ecosistema que aplican, repetidos, más los propios), la tecnología y versiones reales, las convenciones, y las **14 trampas conocidas** que un agente pisa aquí si nadie se lo dice |
| [`arquitectura.md`](arquitectura.md) | La forma por dentro: **un módulo por sección** con qué hace, su versión publicada y quién lo consume; el mapa módulo → consumidores; y la maquinaria de releases |
| [`contratos.md`](contratos.md) | Todo lo que otros consumen: la superficie Go por módulo, las rutas HTTP que este código **llama** (no sirve ninguna), los contratos de datos, y por qué las variables de entorno, los ficheros y los esquemas están **a cero** |
| [`operacion.md`](operacion.md) | Cómo se compila y se prueba, el **procedimiento de release por módulo** con sus tres trampas caras, y cómo se depura cuando falla |
| [`deuda.md`](deuda.md) | La deuda viva con `fichero:linea`, el **código muerto verificado** (nueve símbolos o bloques con cero consumidores) y las contradicciones vivas de los README |

---

## Lo mínimo que hay que saber antes de tocar nada

1. **Los once módulos son de nivel 0: ninguno importa a otro módulo de `wapp-shared`.** Si lo
   rompes, **compila igual**: no hay candado.
2. **El CHANGELOG de un módulo va `## [0.1.0]`, SIN la «v».** Con la «v», el release aborta.
3. **Dar de alta un módulo nuevo toca un tercer registro que vive en OTRO repo**: la línea `use` del
   `go.work` de la raíz `wApp/`. Un `grep` aquí **no puede encontrarlo**.
4. **`ci.yml` es `workflow_dispatch`: un PR no valida nada.** El gate real es `make ci-local` (y
   `make ci-docker`), en tu máquina.
5. **Un consumidor solo está de verdad en verde con `GOWORK=off go build ./...`**, contra el tag
   publicado y no contra el árbol de al lado. **Prohibido el `replace`.**
6. **La DEK nunca cruza hacia la nube en claro** y **la nube nunca ve credenciales ni llaves
   privadas**. `envelope` existe para eso. El contenido de negocio sí sube, y es deliberado.
