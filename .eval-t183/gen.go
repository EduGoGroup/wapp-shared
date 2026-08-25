//go:build ignore

// gen.go — GENERADOR DE PROMPTS del eval de T1.8-3 (Plan 044).
//
// Emite un JSONL con el prompt P1 COMPLETO de cada caso del lote, construido con el MISMO catálogo real
// de UAT y por el MISMO constructor que usa producción. Se ejecuta dos veces —una en la rama y otra con
// el prompt.go anterior— para tener los dos lados del A/B sin que el modelo intervenga: los prompts son
// funciones puras, así que generarlos aquí y sólo INFERIR en el VPS quita una variable del experimento.
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"

	"github.com/EduGoGroup/wapp-shared/llm"
)

type catalogo struct {
	Vocabulario []string `json:"vocabulario"`
	Intents     []struct {
		Name        string   `json:"name"`
		Descripcion string   `json:"descripcion"`
		Params      []string `json:"params"`
		Ejemplos    []struct {
			Mensaje string            `json:"mensaje"`
			Params  map[string]string `json:"params"`
		} `json:"ejemplos"`
	} `json:"intents"`
}

type lote struct {
	Casos []struct {
		Mensaje string `json:"mensaje"`
		Intent  string `json:"intent"`
	} `json:"casos"`
}

func main() {
	if len(os.Args) != 4 {
		fmt.Fprintln(os.Stderr, "uso: gen <catalogo.json> <lote.json> <salida.jsonl>")
		os.Exit(2)
	}
	var cat catalogo
	leer(os.Args[1], &cat)
	var l lote
	leer(os.Args[2], &l)

	// La misma traducción que hace el Cloud (intakeahead.aplanar): si esto divergiera, el eval mediría un
	// prompt que producción nunca construye.
	specs := make([]llm.IntentSpec, 0, len(cat.Intents))
	for _, it := range cat.Intents {
		s := llm.IntentSpec{Name: it.Name, Description: it.Descripcion, Params: it.Params}
		for _, ej := range it.Ejemplos {
			s.Examples = append(s.Examples, llm.IntentExample{Message: ej.Mensaje, Params: ej.Params})
		}
		specs = append(specs, s)
	}

	f, err := os.Create(os.Args[3])
	if err != nil {
		panic(err)
	}
	defer func() { _ = f.Close() }()
	w := bufio.NewWriter(f)
	defer func() { _ = w.Flush() }()

	for i, c := range l.Casos {
		p := llm.BuildClassifyRequestPrompt(llm.ClassifyRequestInput{
			Text:         c.Mensaje,
			Catalog:      specs,
			UnknownLabel: "desconocido",
			Vocabulary:   cat.Vocabulario,
		})
		linea, _ := json.Marshal(map[string]any{"id": i, "esperado": c.Intent, "mensaje": c.Mensaje, "prompt": p})
		_, _ = w.Write(append(linea, '\n'))
		if i == 0 {
			fmt.Fprintf(os.Stderr, "prompt[0] = %d B\n", len(p))
		}
	}
	fmt.Fprintf(os.Stderr, "%d casos escritos en %s\n", len(l.Casos), os.Args[3])
}

func leer(ruta string, v any) {
	b, err := os.ReadFile(ruta)
	if err != nil {
		panic(err)
	}
	if err := json.Unmarshal(b, v); err != nil {
		panic(err)
	}
}
