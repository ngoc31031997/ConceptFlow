# ADR-0012: Content Plugin Integration via Orchestrator (Not Direct REST)

## Status
Accepted

## Date
2026-08-07

## Stage
Low-Level Design (Unit 4: Script Processing Service)

## Context
`component-methods.md` (Application Design) left open how Script Processing Service obtains scene `category` from Content Plugin Service: either a direct internal REST call (mirroring how Rendering Service calls TTS Service), or by staying within the original Saga design (`services.md`), where Parse Script and Classify Scenes are two separate orchestrator-driven steps over AMQP. This had to be resolved before Script Processing Service's interfaces could be finalized.

## Options Considered
### Option A: Direct internal REST call
- What it is: After parsing, Script Processing Service calls Content Plugin Service's REST API synchronously to classify scenes before publishing `script_parsed` (already-classified scenes).
- Strengths: Fewer Saga round-trips; mirrors the Rendering→TTS pattern already built.
- Trade-offs: Content Plugin Service (Unit 2, already code-generated) only exposes classification via an AMQP consumer (`classify_scenes`), not REST — this option would require revisiting an already-"complete" unit to add a new REST endpoint, and it collapses two independently-observable Saga steps into one opaque call, hiding intermediate progress from anything watching Saga events.

### Option B: Via Orchestrator, as originally designed (Chosen)
- What it is: Script Processing Service publishes `script_parsed` with un-classified scenes; Orchestrator (Unit 8) separately dispatches `classify_scenes` to Content Plugin Service as its own Saga step, exactly as `services.md` already specifies.
- Strengths: Matches the approved Saga design exactly — no rework of Unit 2; keeps Script Processing Service decoupled from Content Plugin Service (it doesn't need to know Content Plugin Service exists); each step remains independently observable as a distinct event, which lets Orchestrator persist intermediate results (parsed scenes, then categories) incrementally — directly supporting the project's goal of showing pipeline progress/results in the GUI and avoiding re-work on retry.
- Trade-offs: One additional Saga round-trip before scenes have a category — acceptable since this was the behavior already designed from Application Design onward.

## Decision
Option B — keep Parse Script and Classify Scenes as two separate Saga steps, coordinated by the Orchestrator over AMQP; no direct coupling between Script Processing Service and Content Plugin Service.

## Rationale
This preserves the Saga design already approved in Application Design, avoids reopening a completed unit (Unit 2), and — since each step is a separately observable event — sets up Orchestrator to persist step-level results incrementally, which the user has asked for as a requirement for Unit 8 (GUI visibility into intermediate results, retries that don't redo completed work).

## Consequences
- **Positive**: No changes needed to Unit 2; Script Processing Service stays simple and decoupled; Saga step events remain individually persistable.
- **Negative / Accepted Trade-offs**: Scenes in `script_parsed` lack `category` until the next Saga step completes — GUI/Orchestrator must handle this intermediate state (already anticipated in `services.md`).
- **Follow-ups**: When Unit 8 (Orchestrator) reaches Low-Level Design, its persistence layer should store both `script_parsed` and `scenes_classified` payloads so retries and GUI display don't require re-invoking Script Processing Service or Content Plugin Service for already-completed steps.

## Related
- Design artifact: `aidlc-docs/construction/script-processing-service/low-level-design/interface-contracts.md`
- Related ADRs: Reaffirms the Saga design underlying ADR-0007 (Saga Orchestrator Service + Message Queue)
