package application

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"authoring/internal/domain"
)

// verifierWorkers is how many source checks run at once.
const verifierWorkers = 4

// SourceVerifierStore is where a finished check is written.
type SourceVerifierStore interface {
	// SaveFactLink stores the result only while the fact still has the url and
	// quote the check started from, and reports whether it did.
	SaveFactLink(ctx context.Context, projectID, key, expectedURL, expectedQuote string, link domain.FactLink) (bool, error)
}

type checkJob struct {
	projectID, key, url, quote string
}

// SourceVerifier checks, in the background, that the page a fact cites is
// readable and holds the fact's quote. The caller marks the fact as being
// checked and queues it; the verifier fetches the page, compares the quote and
// writes the result back.
type SourceVerifier struct {
	store   SourceVerifierStore
	fetcher SourceFetcher
	queue   chan checkJob
	done    chan struct{}
	now     func() time.Time
}

// NewSourceVerifier builds a verifier; Start begins its workers.
func NewSourceVerifier(store SourceVerifierStore, fetcher SourceFetcher) *SourceVerifier {
	return &SourceVerifier{
		store:   store,
		fetcher: fetcher,
		queue:   make(chan checkJob),
		done:    make(chan struct{}),
		now:     time.Now,
	}
}

// Start runs the workers until ctx ends. Checks still queued or running then
// are dropped; their facts stay marked as being checked until the next start,
// which turns such marks into an explained failure.
func (v *SourceVerifier) Start(ctx context.Context) {
	go func() {
		<-ctx.Done()
		close(v.done)
	}()
	for i := 0; i < verifierWorkers; i++ {
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case job := <-v.queue:
					v.process(ctx, job)
				}
			}
		}()
	}
}

// Enqueue queues a check of the page projectID's fact key cites. It never
// blocks the caller.
func (v *SourceVerifier) Enqueue(projectID, key, url, quote string) {
	job := checkJob{projectID: projectID, key: key, url: url, quote: quote}
	go func() {
		select {
		case v.queue <- job:
		case <-v.done:
		}
	}()
}

func (v *SourceVerifier) process(ctx context.Context, job checkJob) {
	link, ok := v.check(ctx, job)
	if !ok {
		return
	}
	stored, err := v.store.SaveFactLink(ctx, job.projectID, job.key, job.url, job.quote, link)
	if err != nil {
		slog.Error("save source check failed", "project_id", job.projectID, "fact", job.key, "error", err.Error())
		return
	}
	if !stored {
		slog.Debug("source check dropped: the fact changed meanwhile", "project_id", job.projectID, "fact", job.key)
	}
}

// check fetches the page and compares the quote. ok is false when the check
// was cut short by shutdown, in which case nothing is written.
func (v *SourceVerifier) check(ctx context.Context, job checkJob) (link domain.FactLink, ok bool) {
	stamp := v.now().UTC().Format(time.RFC3339)
	switch {
	case job.url == "":
		return domain.FactLink{State: domain.LinkNone, CheckedAt: stamp}, true
	case strings.TrimSpace(job.quote) == "":
		return domain.FactLink{State: domain.LinkBad, Why: "có link nhưng chưa có đoạn trích để kiểm", CheckedAt: stamp}, true
	}
	page, err := v.fetcher.Fetch(ctx, job.url)
	if err != nil {
		var fe *FetchError
		if errors.As(err, &fe) {
			return domain.FactLink{State: domain.LinkUnreachable, Why: fe.Reason, CheckedAt: stamp}, true
		}
		slog.Warn("source check cut short", "project_id", job.projectID, "fact", job.key, "error", err.Error())
		return domain.FactLink{}, false
	}
	match := domain.MatchQuote(page.Text, job.quote)
	switch {
	case match.Found:
		return domain.FactLink{State: domain.LinkOK, CheckedAt: stamp}, true
	case match.TooShort:
		return domain.FactLink{State: domain.LinkBad, Why: "đoạn trích quá ngắn để kiểm (dưới 20 ký tự)", CheckedAt: stamp}, true
	}
	return domain.FactLink{
		State: domain.LinkBad, Why: "trang tải được nhưng không có đoạn trích này", Nearest: match.Nearest, CheckedAt: stamp,
	}, true
}
