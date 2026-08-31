# Business Logic Summary — Unit 8: Orchestrator Service

## Location
`services/orchestrator/internal/application/`

## Use Cases
| File | Type | Port dependencies |
|---|---|---|
| `start_render_saga.go` | `StartRenderSagaUseCase` | `ProjectRepositoryPort`, `CommandPublisherPort` |
| `start_publish_saga.go` | `StartPublishSagaUseCase` | `ProjectRepositoryPort`, `CommandPublisherPort` |
| `handle_step_event.go` | `HandleStepEventUseCase` | `ProjectRepositoryPort`, `CommandPublisherPort`, `ProgressPublisherPort` |
| `retry_step.go` | `RetryStepUseCase` | `ProjectRepositoryPort`, `CommandPublisherPort` |

Supporting files: `idgen.go` (UUIDv4 generation, no external dependency), `event_payload.go` (pure functions decoding `map[string]interface{}` AMQP payloads into typed `domain.Scene` data).

## event_type → step_name Mapping (`eventStepMap`, handle_step_event.go)
| event_type | step_name |
|---|---|
| `script_parsed` / `parse_failed` | `parse_script` |
| `scenes_classified` / `classification_failed` | `classify_scenes` |
| `speech_synthesized` / `synthesis_failed` | `synthesize_speech` |
| `rendering_completed` / `rendering_failed` | `render_scenes` |
| `video_assembled` / `assembly_failed` | `assemble_video` |
| `video_published` / `publish_failed` | `publish_video` |

`scene_rendered` is not in this map — handled separately as a progress-only event (Rule 7), never advances the state machine.

## Business Rules Implemented
- **Rule 1** (`validateSceneIndexSets`, called from `onSpeechSynthesized`): scene_index sets from `script_parsed`+`scenes_classified` (accumulated on `Project.Scenes`) vs. `speech_synthesized` must match exactly; mismatch → `failed_at_render_scenes` with descriptive `error_message`, no command dispatched, no silent merge.
- **Rule 2** (`assembleVideoPayload`, `onRenderingCompleted`): `audio_path` always read from `Project.Scenes` (set at step 3), never from `rendering_completed`'s payload (which only carries `clip_path`).
- **Rule 3**: `background_music_path` captured once in `StartRenderSagaInput`, stored on `Project` at Saga start, reused unchanged in `assembleVideoPayload`.
- **Rule 4** (`HandleStepEventUseCase.Execute`): `GetStep` checked for `in_progress` before any processing; non-`in_progress` → log warning, skip (return nil so the AMQP consumer still acks), no progress message.
- **Rule 5** (`retry_step.go` `rebuildPayload`): reconstructs any step's payload purely from `Project`, publishes with a new `message_id`.
- **Rule 6**: no concurrency limiting/locking in this layer — enforced structurally by keying all operations on `project_id`.
- **Rule 7**: `ProgressMessage` published after every successful/failed/skip-exempt processing, including `scene_rendered`.
- **Rule 8**: failure handling (`handleFailure`) only transitions status — never deletes or reverts `Project.Scenes` data.
- **Rule 9**: Inbox/Outbox semantics implemented in `internal/adapters/postgres` (`inbox.go`, `outbox.go`), not in this layer — this layer only calls the `CommandPublisherPort` abstraction.

## Testing
Each use case has a `_test.go` using hand-written in-memory fakes (`fakes_test.go`: `fakeRepo`, `fakePublisher`, `fakeProgress`) implementing the 3 domain ports — no real Postgres/RabbitMQ, no mocking library.
