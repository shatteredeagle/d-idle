---
status: accepted
date: 2026-10-29
decision-makers: user (decisions made in conversation), coordinator
---
# 0001: The end goal is a container; native on the worker host stays the dev path

## Context and Problem Statement

We need one supported way for other people to run D.idle on Linux, NAS and Windows or Intel Mac, while agents still need fast iteration without container rebuilds.

## Considered Options

* Container only
* Native only
* Container as product, native as dev mode

## Decision Outcome

Container as product, native on the worker host as an internal dev mode. No native-only user features; new runtime behaviour must work in the container (explicit mounts only). The container is verified with the same stamped binaries after the 24 h soak, by a non-agent. No Windows-native port.

## Consequences

Wine container work was later paused by ADR 0002.

## Sources

decisions.md 2026-10-29
