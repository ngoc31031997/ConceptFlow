package domain

import "errors"

// Sentinel errors returned by use cases and repository adapters. Adapters
// (e.g. internal/adapters/http) translate these to the appropriate transport
// status code (404, 409) rather than leaking infrastructure error types.
var (
	// ErrProjectNotFound is returned when a project_id has no matching Project.
	ErrProjectNotFound = errors.New("project not found")

	// ErrUnexpectedEvent is returned (and logged as a warning, not an error to
	// the caller) when an event's step is not currently in_progress —
	// business-rules.md Rule 4 / sequence-flows.md Flow 6.
	ErrUnexpectedEvent = errors.New("unexpected event for non-in-progress step")

	// ErrInvalidStatus is returned when a REST call's precondition on
	// Project.Status is not met (e.g. POST /v1/sagas/publish when status is
	// not ready_to_publish, or POST /retry when status is not failed_at_*).
	ErrInvalidStatus = errors.New("project status does not allow this operation")

	// ErrSagaStepNotFound is returned when GetStep finds no matching SagaStep.
	ErrSagaStepNotFound = errors.New("saga step not found")
)
