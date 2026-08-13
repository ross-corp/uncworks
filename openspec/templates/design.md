# <Change title> design

> The keywords MUST, MUST NOT, SHOULD, SHOULD NOT, and MAY in this document are
> to be interpreted as described in [RFC 2119](https://datatracker.ietf.org/doc/html/rfc2119).

## Context

<Current state, constraints, and the existing patterns being extended. Cite
paths. If you introduce a new pattern, say why the closest existing one was
insufficient.>

## Goals / Non-Goals

Goals:

- <What this design achieves.>

Non-goals:

- <What it explicitly excludes.>

## Decisions

- Decision: <the choice, and why.>
- Alternative rejected: <what lost, and the reason it lost.>

## Rollout & Gating

<Which phase ships first, what gate must pass before the next, and where the
kill switch is.>

1. <Phase 1>, gated on `<command>` exiting 0.
2. <Phase 2>, gated on <the owner's judgment>.

Kill switch: <flag, toggle, or revert path.>

## Risks / Trade-offs

- <Risk> mitigated by <mitigation>.

## Migration Plan

<Deploy steps and rollback. Every step that mutates shared state is preceded by
a verification step and followed by a confirmation step.>

## Open Questions

- <Unresolved question, and who decides it.>
