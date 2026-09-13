# a3-go

CORE implementation of SPEC_A3-EP 0.2.0 in Go. Written from the spec and lock vectors, not from the Kotlin or TypeScript sources.

Profile: **CORE**. Extensions not implemented: overlay, agent, broker, sensory.

## Run

```
go test ./...
```

Lock vectors live in `conformance/vectors/v2/` (byte copies of a3 lock v2) and `conformance/vectors/v1/` (backward parse). Category fixtures in `conformance/fixtures/`.

## CORE

| Package | Spec |
|---------|------|
| `jcs` | RFC 8785 canonical JSON |
| `envelope` | CloudEvents 1.0, `id` = SHA-256(JCS(payload)), type registry, attestation |
| `temporal` | four instants, order `(t_observe, source_id, seq)`, append-only log, projection without folding history |
| `truth` | FACT / OBSERVATION / HYPOTHESIS / UNKNOWN + provenance + TR-001..006 |
| `confidence` | five-dimension weighted min, recency `2^(-Δt/21600)`, corroboration, mapping |

Runtime dependencies: Go 1.22+ standard library only (`crypto/sha256`, `math`, `encoding/json`). Generics are native; `golang.org/x/exp` is not required.

## event_id

| Lock | SHA-256(JCS(payload)) |
|------|------------------------|
| v1 | `477e868489f5c48d138e4c084e9bf13a40ed66390b964365869f7578dfa2e75a` |
| v2 | `1fec213fbaf6d420cf9ff95c51c022c4cdfb1f43fabcf82647e03c03f92f2b7b` |

v2 `type` is `io.a3ep.belief.admitted`. Parser accepts `a3.*` and normalizes. Output does not emit `a3.*`.

## Add a check

Keep `conformance/vectors/v2/` byte-identical. Hash against `vector-sha256.json`. Reject CF-001..CF-009 with `reject.Error` whose `Code` is `CF-00N`. Do not admit a producer that violates a MUST.

## Report

```
implementation: a3-go
language: Go
version: a3-go-v0.1
profile: CORE
extensions: none
vectors: SHA-256 match vector-sha256.json: yes
```
