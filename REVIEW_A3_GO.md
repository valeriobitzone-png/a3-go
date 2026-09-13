# REVIEW_A3_GO

AUDIT-FIRST. Protocol: FASE A3-GO — terza implementazione CORE da SPEC_A3-EP 0.2.0. Repo nuovo `a3-go`, privato SSH. Niente push prima del tag. Tag `a3-go-v0.1` solo a gate verde; poi push.

**Contratto:** implementazione da spec + lock v2, non traduzione di Kotlin o TypeScript. Un producer che viola un MUST fallisce. Collisione tripla sui byte di `envelope-event.json` / `envelope-payload.json` / `event_id` v2.

---

## Identità lock v2

| | |
|--|--|
| `event_id` | `1fec213fbaf6d420cf9ff95c51c022c4cdfb1f43fabcf82647e03c03f92f2b7b` |
| `type` | `io.a3ep.belief.admitted` |
| attestation | `attester_id=urn:a3:party:attester`, `requester_id=urn:a3:party:requester` |

event_id v1 (backward parse): `477e868489f5c48d138e4c084e9bf13a40ed66390b964365869f7578dfa2e75a`.

---

## SHA-256 vettori v2 (verificati)

| File | SHA-256 |
|------|---------|
| tm-order.json | `e805711fc39d48a59b47bfdd147737016db56a3a68511f27229e769691378a6e` |
| tm-dedup.json | `b6b05fefb45b1f9ff2fc882d16eb5a1f0b1cf96e6d8455070836d472e333e0ba` |
| tm-fold.json | `f80b9b513fc928d11e8aceb66a29d7cfb7540a0bb8451c9017630b105602e9f5` |
| truth-vectors.json | `1ddb48779a470fd65adc59a5e0245767f07bd4ea91ad70b7c3e10afbdebab5d6` |
| envelope-rfc8785.json | `2d5e01a318d0f0879ab568c4be289c8b1f64ef8921a53c6277d5e069978baacb` |
| envelope-payload.json | `1fec213fbaf6d420cf9ff95c51c022c4cdfb1f43fabcf82647e03c03f92f2b7b` |
| envelope-event.json | `fa6e007a23751ad55c22291b64982f0d7c8287eb5723b446a3a5fd72c470e939` |
| envelope-hashes.json | `d27e67f05719b77daeb14a4d219a87cb332998fe2c6d163571f35e7b90d76ff5` |
| confidence-vectors.json | `25efbc9f1b3730658c34502f2564d18a8aad04c1ee202b672be4b020917fddee` |
| event_id_expected.txt | `c7cb220cb548ecb3be575faecac6d0d7e57cc18a785727732e5570c21bb68550` |

---

## Esecuzione

```
go test ./...
```

---

## Tabella audit — GO-001..009

| Test | Invariante | Percorso | PASS |
|------|------------|----------|------|
| GO-001 | RFC 8785 appendix A byte-identico | `jcs.Canonicalize(rfc8785-input.json)` = `envelope-rfc8785.json` | **PASS** |
| GO-002 | envelope lock v2; `id` = `1fec213f…` | Pack + parse `envelope-event.json`; CF-008 | **PASS** |
| GO-003 | tm-order / dedup / fold = lock v2 | `temporal.Sort/Dedup/Project`; CF-006, CF-009 | **PASS** |
| GO-004 | truth-vectors + reject TR/CF | `truth.LockVectors`; CF-001/002/003/005 | **PASS** |
| GO-005 | confidence scores = lock | weighted min on lock vector; CF-007 | **PASS** |
| GO-006 | Go serialize = canonico Kotlin/TS v2 | JCS(event) = `envelope-event.json` | **PASS** |
| GO-007 | parse v1 `a3.*` → `io.a3ep.*` | `envelope.Parse` su `vectors/v1/envelope-event.json` | **PASS** |
| GO-008 | attester_id = requester_id su irreversibile → reject | CF-004 + `PackV2(action.authorized)` | **PASS** |
| GO-009 | a3 e a3-ts non modificati | `git diff --stat` vuoto su entrambi | **PASS** |

Gate: **verde**. Tag `a3-go-v0.1`. Push dopo il tag.

---

## Divergenza dichiarata (recency)

SPEC_A3-EP §7: recency = `2^(-(t_present - t_observe) / half_life)` con half-life 21600s.

Il lock memorizza `fixture_10799s = 0.7071294727113612` (Kotlin: IEEE-754 nearest-even sul valore scritto). `math.Pow(2, -10799/21600)` in Go 1.22+ su questa macchina produce `0.7071294727113613` (1 ULP).

Non è stato applicato un arrotondamento ad hoc per coincidere col lock. Lo **score** del vettore lock (`0.565703578169089` = recency_lock × 0.8) è identico perché GO-005 aggrega il vettore dichiarato, non il Pow dal Δt. Nessun altro byte del lock v2 diverge.

CPython 3 sullo stesso host dà `0.7071294727113612`; la differenza è libm Go vs valore memorizzato, non un bug di weighted min.

---

## Fuori dal CORE (non implementato)

Overlay/lifecycle, agent, broker, sensory/motion/glass.

---

## Vietato (rispettato)

- Nessuna traduzione di sorgenti Kotlin o TypeScript.
- Nessuna dipendenza runtime oltre la stdlib.
- Nessun modulo fuori dal CORE.
- Divergenza recency dichiarata, non nascosta.
- Repo a3 e a3-ts non toccati.
- Nessun push prima del tag.
