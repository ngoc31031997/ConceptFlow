package domain

import "time"

// QCStatus is the overall verdict of one automated QC pass (CR-021 D6).
//
// `not_scored` is a first-class verdict, not an error: FR61.4 says a QC that
// could not run (marks missing, ffprobe failed) must report that it could not
// score the video and let the publish proceed. Folding it into "passed" would
// hide a broken measurement behind a green light; folding it into a failure
// would turn a broken gate into a locked one.
type QCStatus string

const (
	QCStatusPassed      QCStatus = "passed"
	QCStatusHasFindings QCStatus = "has_findings"
	QCStatusNotScored   QCStatus = "not_scored"
)

// QC finding severities. Only Blocking is ever capable of stopping a publish,
// and even then only with QC_ENFORCE on (CR-021 D5 — the indicate-first mode).
const (
	QCSeverityBlocking = "blocking"
	QCSeverityWarning  = "warning"
)

// QCFinding is one thing the QC pass noticed, shaped exactly as the
// `qc_completed` event carries it (CR-021 D4).
//
// TimestampSeconds is what makes a report actionable rather than a list of
// complaints: FR61.2's GUI turns it into a click that seeks the <video> to the
// moment in question.
type QCFinding struct {
	Rule             string  `json:"rule"`
	Severity         string  `json:"severity"`
	Message          string  `json:"message"`
	TimestampSeconds float64 `json:"timestamp_seconds"`
}

// QCReport is one stored QC pass for a project (FR61.1 — machine-readable,
// not a log line).
//
// OverriddenAt/OverriddenFindings record a deliberate bypass (FR61.3). They
// live on the report rather than on the project because what matters later is
// *which* findings were waved through, and that is only meaningful next to the
// report they came from.
type QCReport struct {
	ProjectID           string
	Status              QCStatus
	Reason              *string
	Findings            []QCFinding
	CreatedAt           time.Time
	OverriddenAt        *time.Time
	OverriddenFindings  []QCFinding
}

// BlockingFindings returns only the findings severe enough to stop a publish.
func (r *QCReport) BlockingFindings() []QCFinding {
	if r == nil {
		return nil
	}
	out := make([]QCFinding, 0, len(r.Findings))
	for _, f := range r.Findings {
		if f.Severity == QCSeverityBlocking {
			out = append(out, f)
		}
	}
	return out
}

// HasBlockingFindings reports whether this report would stop a publish under
// QC_ENFORCE.
func (r *QCReport) HasBlockingFindings() bool {
	return len(r.BlockingFindings()) > 0
}
