# Changelog — textmatch

El formato sigue [Keep a Changelog](https://keepachangelog.com/es-ES/1.0.0/)
y [Versionado Semantico](https://semver.org/lang/es/).

## [Unreleased]

## [0.1.0] - 2026-08-26

### Added

- Version inicial del modulo `textmatch`: motor determinista de comparacion de
  textos, copia-adaptacion de `edugo-shared/textmatch` (ADR-0004) para el
  Plan 044 · Ola 3 (T3.1).
  - `Normalize` / `SplitTokens`: normalizacion canonica (minusculas, sin tildes ni
    dieresis, espacios colapsados) que **preserva la ñ**. El plegado de
    diacriticos es propio y de stdlib: se retiro la dependencia
    `golang.org/x/text` del original.
  - `EditDistance`: Damerau-Levenshtein restringida (OSA) por runas; la
    transposicion adyacente cuenta como UNA edicion.
  - Nivel 1: `Outcome`, `Result`, `Strategy`, `Comparator`, `Exact`,
    `Fuzzy`/`NewFuzzy` (umbral por defecto `DefaultFuzzyThreshold` = 0,85) y
    `Cascade`/`NewCascade` con escalado explicito (positivo corta, negativo
    escala, error se propaga).
  - Nivel 2: `Policy` (`PolicyStrict`/`PolicyLenient`), `Candidate`,
    `GenerateCandidates`, `MatchReport` y `SetMatcher` con `Match` (candidatos
    atomicos) y `MatchAnswer` (texto crudo + n-gramas).
  - Zona gris **inyectada** (DIP): interfaz `GrayZone` + `GrayZoneDecision`,
    `Cascade.WithGrayZone` / `Cascade.Deterministic` / `Cascade.HasGrayZone` y
    `SetMatcher.WithGrayZone`. El modulo **no importa** `wapp-shared/llm`. Sin
    zona gris cableada el motor es determinista puro, y en el `SetMatcher` el
    escalon caro se consulta fuera del bucle: como mucho una vez por esperado sin
    cubrir.
  - Sin dependencias de produccion; `github.com/stretchr/testify` solo en tests.
