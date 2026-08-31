# API Layer Summary — Unit 8: Orchestrator Service

## Location
`services/orchestrator/internal/adapters/http/`

## Router
`chi.NewRouter()`-based `Router` (router.go), constructed via `NewRouter(startRenderSaga, startPublishSaga, retryStep, projects)` — 4 narrow local interfaces (one per handler's use case dependency), not the full `application` structs, so `router_test.go` can supply small fake use cases without depending on `internal/application`'s fakes.

## Routes
| Method | Path | Use case | Success | Errors |
|---|---|---|---|---|
| GET | `/health` | — | 200 `{"status":"ok"}` | — |
| POST | `/v1/sagas/render` | `StartRenderSagaUseCase` | 201 `{saga_id, status}` | 400 (missing/invalid fields) |
| POST | `/v1/sagas/publish` | `StartPublishSagaUseCase` | 201 `{saga_id, status}` | 400, 409 (`ErrInvalidStatus` → project not `ready_to_publish`) |
| GET | `/v1/projects/{project_id}` | `ProjectRepositoryPort.Get` | 200 full project | 404 (`ErrProjectNotFound`) |
| POST | `/v1/projects/{project_id}/retry` | `RetryStepUseCase` | 200 `{saga_id, status}` | 409 (`ErrInvalidStatus` → project not `failed_at_<step>`) |

## Error Mapping
`writeUseCaseError` (router.go) translates domain sentinel errors to HTTP status via `errors.Is`: `ErrProjectNotFound` → 404, `ErrInvalidStatus` → 409, anything else → 500. Input validation errors (missing required fields, invalid enum values) are checked before calling the use case and return 400 directly.

## DTOs (dto.go)
Plain structs + `encoding/json` tags, no schema library. `toProjectResponse` maps `domain.Project`/`domain.Scene` to the wire shape (`projectResponse`/`sceneResponse`).

## Testing
`router_test.go` uses `net/http/httptest` with fake use cases (`fakeStartRenderSaga`, `fakeStartPublishSaga`, `fakeRetryStep`, `fakeProjectReader`) — covers 201/400/409/404/200 paths and `/health`.
