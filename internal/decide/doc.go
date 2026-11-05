// Package decide is the offline decision-point store of the automation ladder (ADR 0016, spec
// docs/research/decision-point-store-spec.md, TASK-19).
//
// The catalogue (catalogue.json, embedded) is data: points, answers, banded context fields, the risk table by
// kit and hardcore flag, and the untrained default per point. A call site builds a Context from banded fields and
// calls Store.Ask: a learned rule answers at once, otherwise the default is returned at once and a thought opens
// that fades unanswered (virtual Clock). The player's Answer feeds a streak per context key; N identical answers
// in a row (2/3/5 by risk class) teach a rule, risky points confirm first. Rules never revoke themselves.
//
// Every state change is a line of the append-only log; rules are a pure function of the log (Replay). Profiles
// persist atomically with a version field and keep rules, log and quality through a rebirth.
//
// No class or skill name appears in Go; build numbers live in the catalogue's class table. Nothing here touches
// game memory.
package decide
