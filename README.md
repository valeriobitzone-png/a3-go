# a3-go — A3-EP Go reference

![a3-go cover](docs/assets/cover.png)

A Go implementation of A3-EP v0.2.0 written from the normative specification and lock vectors.

[![License](https://img.shields.io/badge/license-Apache--2.0-blue.svg)](LICENSE) [![Latest tag](https://img.shields.io/github/v/tag/valeriobitzone-png/a3-go?sort=semver)](https://github.com/valeriobitzone-png/a3-go/tags) [![CI](https://github.com/valeriobitzone-png/a3-go/actions/workflows/ci.yml/badge.svg)](https://github.com/valeriobitzone-png/a3-go/actions/workflows/ci.yml)

## What it is

A standard-library Go implementation of JCS, CloudEvents, temporal ordering, truth, confidence, provenance, and explicit rejection.

## What it is NOT

It is not a UI renderer, not a port of another implementation, and not a claim of production deployment or numerical identity outside the lock corpus.

## Status

- **VERIFIED:** `go test ./...`, lock-vector canonical bytes, category rejects, and standard-library boundaries.
- **UNVERIFIED:** third-party adoption and physical deployment.

## Quickstart

### Get it

```bash
git clone https://github.com/valeriobitzone-png/a3-go.git
cd a3-go
# Requirements: Go 1.22+.
git clone https://github.com/valeriobitzone-png/a3.git ../a3
git -C ../a3 checkout closeout-v1.0
git clone https://github.com/valeriobitzone-png/a3-ts.git ../a3-ts
git -C ../a3-ts checkout prepub-v1.0
```

### Prove it

```bash
go test ./...
```

### Integrate it

```bash
go get github.com/valeriobitzone-png/a3-go@main
```

```go
import "github.com/valeriobitzone-png/a3-go/truth"

// whatever your tool answered is a receipt: OBSERVATION, never FACT
obs, _ := truth.AdmitOnReceipt(truth.OBSERVATION, truth.Receipt{Payload: result}, "tool:send_email")
_, err := truth.AdmitOnReceipt(truth.FACT, truth.Receipt{Payload: result}, "") // reject CF-001

// FACT only from a verification of the world in REAL
fact, _ := truth.Promote(
    truth.Bearer{Class: truth.HYPOTHESIS, Provenance: truth.DerivedModel},
    &truth.Verification{VerificationID: "ver-1", AdmittedID: "outbox:msg-1", Environment: truth.REAL},
)
```

There is no semver release tag yet, so `go get` resolves a pseudo-version. The normative text is `../a3/spec/SPEC_A3-EP.md`; the conformance tests show every reject code in use.

## Architecture

Protocol concerns are separated into `envelope/`, `jcs/`, `temporal/`, `truth/`, `confidence/`, `reject/`, and `conformance/`. Runtime dependencies are standard library only.

## Testing & conformance

CI runs `go test ./...`. The conformance package compares canonical bytes and rejection behavior with the shared lock corpus.

## Family

- [a3](https://github.com/valeriobitzone-png/a3) — normative specs and Kotlin implementation
- [a3-ts](https://github.com/valeriobitzone-png/a3-ts) — TypeScript reference implementation
- [a3ui-web](https://github.com/valeriobitzone-png/a3ui-web) — Web Components renderer
- [a3ui-cli](https://github.com/valeriobitzone-png/a3ui-cli) — textual Python renderer
- [a3ui-graphics](https://github.com/valeriobitzone-png/a3ui-graphics) — renderer-neutral graphics tokens

## Contributing

Keep protocol/spec boundaries explicit, preserve provenance, and add regression evidence for behavior changes.

## License

Implementation code is Apache-2.0. The governing specification and schemas are CC BY 4.0 in the pinned A3 repository.

## Provenance

Measured: Go tests, canonical lock bytes, and reject cases. Implementation differences outside the lock corpus are not inferred to be equivalent.
