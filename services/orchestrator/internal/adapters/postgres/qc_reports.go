package postgres

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"orchestrator/internal/domain"
)

// QCReportRepository implements domain.QCReportPort against the qc_reports
// table (CR-021 D6).
type QCReportRepository struct {
	pool *pgxpool.Pool
}

// NewQCReportRepository constructs the repository over an already-open pool.
func NewQCReportRepository(pool *pgxpool.Pool) *QCReportRepository {
	return &QCReportRepository{pool: pool}
}

// SaveQCReport inserts one completed QC pass.
//
// Append-only rather than upsert: a project re-rendered after a fix is scored
// again, and the previous report describes a video file that no longer exists —
// overwriting it would destroy exactly the before/after pair that makes
// threshold calibration (D5) possible. Duplicate inserts from a redelivered
// event are prevented upstream by the Inbox (processed_messages), which is the
// one place in this service that dedupes incoming events.
func (r *QCReportRepository) SaveQCReport(ctx context.Context, report domain.QCReport) error {
	findings := report.Findings
	if findings == nil {
		findings = []domain.QCFinding{}
	}
	findingsJSON, err := json.Marshal(findings)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx, `
		INSERT INTO qc_reports (project_id, status, reason, findings, created_at)
		VALUES ($1, $2, $3, $4, now())`,
		report.ProjectID, string(report.Status), report.Reason, findingsJSON)
	return err
}

// LatestQCReport returns the newest report for a project, or (nil, nil) when
// there is none.
//
// The nil-not-error contract matters: every project rendered before CR-021 has
// no report, and the publish gate must read that as "nothing to enforce" rather
// than as a failure — the same reasoning FR61.4 applies to an unscorable video.
func (r *QCReportRepository) LatestQCReport(ctx context.Context, projectID string) (*domain.QCReport, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT project_id, status, reason, findings, created_at, overridden_at, overridden_findings
		FROM qc_reports WHERE project_id = $1 ORDER BY created_at DESC LIMIT 1`, projectID)

	var (
		report                          domain.QCReport
		status                          string
		findingsJSON, overriddenJSON    []byte
	)
	err := row.Scan(&report.ProjectID, &status, &report.Reason, &findingsJSON,
		&report.CreatedAt, &report.OverriddenAt, &overriddenJSON)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	report.Status = domain.QCStatus(status)
	if len(findingsJSON) > 0 {
		if err := json.Unmarshal(findingsJSON, &report.Findings); err != nil {
			return nil, err
		}
	}
	if len(overriddenJSON) > 0 {
		if err := json.Unmarshal(overriddenJSON, &report.OverriddenFindings); err != nil {
			return nil, err
		}
	}
	return &report, nil
}

// RecordQCOverride stamps the project's latest report as deliberately bypassed
// (FR61.3).
//
// `AND overridden_at IS NULL` is what makes a second press of the publish
// button harmless: the recorded moment is the first conscious decision, not the
// last retry of it — the same idempotency shape CR-024's review gate uses.
func (r *QCReportRepository) RecordQCOverride(ctx context.Context, projectID string, findings []domain.QCFinding) error {
	if findings == nil {
		findings = []domain.QCFinding{}
	}
	findingsJSON, err := json.Marshal(findings)
	if err != nil {
		return err
	}
	_, err = r.pool.Exec(ctx, `
		UPDATE qc_reports SET overridden_at = now(), overridden_findings = $2
		WHERE project_id = $1 AND overridden_at IS NULL
		  AND created_at = (SELECT max(created_at) FROM qc_reports WHERE project_id = $1)`,
		projectID, findingsJSON)
	return err
}
