# ADR-0014: TTS Service Becomes Message-Driven (Own Saga Step)

## Status
Accepted

## Date
2026-08-07

## Stage
Cross-cutting retrofit (initiated during Unit 4 Low-Level Design); supersedes the TTS interface decisions made in Application Design (`component-methods.md`) and Unit 3's original Low-Level Design/NFR Design.

## Context
TTS Service was originally designed as the one unit with no RabbitMQ participation: Rendering Service calls it synchronously via REST (`POST /v1/tts/synthesize`) *inside* its own Saga step (`render_scenes`), because Rendering needs `duration_seconds` immediately to synchronize animation timing (FR4.3). This was a deliberate, reasonable choice at the time (see Unit 3's NFR Design/NFR Requirements — "Saga role: indirect participant, no compensating action needed").

Retrofitting Inbox/Outbox onto TTS Service requires it to actually consume/publish messages — Inbox/Outbox is a messaging pattern, and applying it to a purely synchronous REST service would mean bolting on unused messaging infrastructure. The user confirmed (via AskUserQuestion) that they want TTS Service converted to fully message-driven, becoming its own Saga step, rather than staying REST-only.

## Options Considered
### Option A: AMQP Request-Reply (RPC pattern) — kept Saga step boundary unchanged
- What it is: Rendering Service still owns the `render_scenes` Saga step; internally, it publishes a `synthesize_speech` command with a reply-to queue + correlation ID and awaits the `speech_synthesized` reply, instead of a REST call. `services.md`'s Saga Step Definitions table stays unchanged.
- Strengths: No change to the approved Saga step structure; TTS gains a real Inbox (dedupe by message_id) and Outbox (write result + publish reply atomically).
- Trade-offs: Still logically synchronous from Rendering's perspective (it blocks waiting for the reply) — a smaller step toward "fully" message-driven; more complex to implement correctly (correlation ID matching, reply queue lifecycle) than a plain broadcast event.

### Option B: TTS becomes its own top-level Saga step, orchestrated directly (Chosen)
- What it is: `services.md`'s Saga Step Definitions gains a new step "Synthesize Speech" between "Classify Scenes" and "Render Scenes." Orchestrator dispatches `synthesize_speech` (per scene) directly to TTS Service; TTS Service publishes `speech_synthesized`/`synthesis_failed` back to `orchestrator.events`; Orchestrator then dispatches `render_scenes` (now animation-only, consuming already-produced audio paths/durations) once all scenes for a project have synthesized audio.
- Strengths: Fully decouples TTS Service from Rendering Service (no service-to-service call at all — matches pure microservices/Saga orchestration style, the user's stated learning goal); each step remains independently observable/persistable as a distinct event, directly supporting the separate persistence goal already recorded in `project_orchestrator_persistence.md`.
- Trade-offs: Larger change — `services.md`, `component-methods.md`, and `unit-of-work.md` all need updating; Rendering Service's design (not yet built) must assume audio is already available rather than fetching it itself; Orchestrator must now coordinate a new step and determine when all per-scene synthesis for a project has completed before advancing.

## Decision
Option B — TTS Service becomes its own Saga step, coordinated by the Orchestrator, publishing a completion event; Rendering Service no longer calls TTS at all.

## Rationale
The user explicitly chose this over the RPC-preserving option, prioritizing full decoupling and independently observable/persistable Saga steps over minimizing change to the already-approved Saga shape — consistent with the project's learning-first goal and the standing requirement that Orchestrator persist each step's actual result (`project_orchestrator_persistence.md`).

## Consequences
- **Positive**: TTS Service is now a normal message-driven participant like every other business service, eligible for the same Inbox/Outbox pattern (ADR-0013); Saga progress becomes more granular/observable (a distinct "audio synthesized" milestone per scene, not hidden inside Rendering's step).
- **Negative / Accepted Trade-offs**: Rendering Service (Unit 5, not yet designed) must be designed around already-available audio rather than fetching it itself — a constraint to carry into its own Low-Level Design; Orchestrator (Unit 8) must handle per-scene fan-out/fan-in for this step (dispatch N `synthesize_speech` commands for N scenes, wait for all N `speech_synthesized` events before advancing) — added complexity acknowledged as a Unit 8 design concern, not solved here.
- **Follow-ups**: Update `services.md` (new Saga step + sequence diagram), `component-methods.md` (TTS interface section), `unit-of-work.md` (Unit 3 now depends on Unit 1). Unit 3's Low-Level Design, Functional Design, NFR Requirements/Design, Infrastructure Design, and Code Generation all need revisiting to reflect the new AMQP-based interface.

## Related
- Design artifact: `aidlc-docs/inception/application-design/services.md` (to be updated), `aidlc-docs/construction/tts-service/`
- Related ADRs: Supersedes the TTS interface framing implicit in Application Design's original `component-methods.md`; reaffirms ADR-0007 (Saga Orchestrator Service + Message Queue) as the consistent pattern for all business services
