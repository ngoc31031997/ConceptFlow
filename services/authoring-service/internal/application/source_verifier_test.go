package application

import (
	"context"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"authoring/internal/domain"
)

// fakeFetcher serves fixed pages and records every call. When gate is set each
// fetch waits on it, so a test can hold downloads open.
type fakeFetcher struct {
	mu      sync.Mutex
	pages   map[string]string
	errs    map[string]error
	calls   []string
	gate    chan struct{}
	running atomic.Int32
	peak    atomic.Int32
}

func (f *fakeFetcher) Fetch(ctx context.Context, url string) (SourcePage, error) {
	n := f.running.Add(1)
	defer f.running.Add(-1)
	for {
		p := f.peak.Load()
		if n <= p || f.peak.CompareAndSwap(p, n) {
			break
		}
	}
	f.mu.Lock()
	f.calls = append(f.calls, url)
	page, hasPage := f.pages[url]
	err := f.errs[url]
	gate := f.gate
	f.mu.Unlock()
	if gate != nil {
		select {
		case <-gate:
		case <-ctx.Done():
			return SourcePage{}, ctx.Err()
		}
	}
	if err != nil {
		return SourcePage{}, err
	}
	if !hasPage {
		return SourcePage{}, &FetchError{Reason: "không tìm thấy trang (404)"}
	}
	return SourcePage{Text: page}, nil
}

func (f *fakeFetcher) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.calls)
}

// linkRecorder keeps what a check wrote.
type linkRecorder struct {
	mu     sync.Mutex
	saved  []domain.FactLink
	stored bool
	err    error
}

func (r *linkRecorder) SaveFactLink(_ context.Context, _, _, _, _ string, link domain.FactLink) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.saved = append(r.saved, link)
	return r.stored, r.err
}

func (r *linkRecorder) last() domain.FactLink {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.saved) == 0 {
		return domain.FactLink{}
	}
	return r.saved[len(r.saved)-1]
}

const checkQuote = "The female wasp loses her wings as she squeezes through the ostiole."

func TestSourceVerifierResults(t *testing.T) {
	page := "Intro. The female wasp loses her wings as she squeezes through the ostiole. More text."
	fetcher := &fakeFetcher{
		pages: map[string]string{
			"https://a.org/ok":    page,
			"https://a.org/other": "The relationship is thought to have existed for sixty million years. Nothing else here.",
		},
		errs: map[string]error{"https://a.org/paywall": &FetchError{Reason: "trang trả phí hoặc cần đăng nhập (HTTP 403)"}},
	}
	cases := []struct {
		name    string
		job     checkJob
		state   domain.LinkState
		why     string
		nearest string
	}{
		{"quote found", checkJob{url: "https://a.org/ok", quote: checkQuote}, domain.LinkOK, "", ""},
		{"page without the quote", checkJob{url: "https://a.org/other", quote: "The relationship is believed to have existed for sixty million years."}, domain.LinkBad, "không có đoạn trích", "thought to have existed"},
		{"page unreachable", checkJob{url: "https://a.org/paywall", quote: checkQuote}, domain.LinkUnreachable, "trả phí", ""},
		{"missing page", checkJob{url: "https://a.org/gone", quote: checkQuote}, domain.LinkUnreachable, "404", ""},
		{"no link", checkJob{quote: checkQuote}, domain.LinkNone, "", ""},
		{"link without quote", checkJob{url: "https://a.org/ok"}, domain.LinkBad, "chưa có đoạn trích", ""},
		{"quote too short", checkJob{url: "https://a.org/ok", quote: "wings"}, domain.LinkBad, "quá ngắn", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rec := &linkRecorder{stored: true}
			v := NewSourceVerifier(rec, fetcher)
			v.process(context.Background(), c.job)
			got := rec.last()
			if got.State != c.state || got.CheckedAt == "" {
				t.Fatalf("link = %+v, want state %q with a time", got, c.state)
			}
			if c.why != "" && !strings.Contains(got.Why, c.why) {
				t.Errorf("Why = %q, want it to mention %q", got.Why, c.why)
			}
			if c.nearest != "" && !strings.Contains(got.Nearest, c.nearest) {
				t.Errorf("Nearest = %q, want it to mention %q", got.Nearest, c.nearest)
			}
		})
	}
}

func TestSourceVerifierWritesNothingWhenCutShort(t *testing.T) {
	rec := &linkRecorder{stored: true}
	fetcher := &fakeFetcher{errs: map[string]error{"https://a.org/x": context.Canceled}}
	NewSourceVerifier(rec, fetcher).process(context.Background(), checkJob{url: "https://a.org/x", quote: checkQuote})
	if len(rec.saved) != 0 {
		t.Fatalf("a check cut short by shutdown must write nothing, wrote %+v", rec.saved)
	}
}

func TestSourceVerifierSurvivesAStoreFailure(t *testing.T) {
	rec := &linkRecorder{err: errors.New("db down")}
	fetcher := &fakeFetcher{pages: map[string]string{"https://a.org/ok": checkQuote}}
	NewSourceVerifier(rec, fetcher).process(context.Background(), checkJob{url: "https://a.org/ok", quote: checkQuote})
	if len(rec.saved) != 1 {
		t.Fatalf("the result must still be offered to the store once, got %d", len(rec.saved))
	}
}

func TestSourceVerifierRunsAtMostFourChecksAtOnce(t *testing.T) {
	gate := make(chan struct{})
	fetcher := &fakeFetcher{pages: map[string]string{"https://a.org/ok": checkQuote}, gate: gate}
	rec := &linkRecorder{stored: true}
	v := NewSourceVerifier(rec, fetcher)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	v.Start(ctx)
	for i := 0; i < 10; i++ {
		v.Enqueue("p1", "F01", "https://a.org/ok", checkQuote)
	}
	waitUntil(t, func() bool { return fetcher.running.Load() == verifierWorkers })
	time.Sleep(50 * time.Millisecond)
	if got := fetcher.running.Load(); got != verifierWorkers {
		t.Fatalf("running downloads = %d, want exactly %d while the rest wait", got, verifierWorkers)
	}
	close(gate)
	waitUntil(t, func() bool {
		rec.mu.Lock()
		defer rec.mu.Unlock()
		return len(rec.saved) == 10
	})
	if fetcher.peak.Load() > verifierWorkers {
		t.Fatalf("peak concurrent downloads = %d, want at most %d", fetcher.peak.Load(), verifierWorkers)
	}
}

func TestSourceVerifierStopsWithItsContext(t *testing.T) {
	gate := make(chan struct{})
	fetcher := &fakeFetcher{pages: map[string]string{"https://a.org/ok": checkQuote}, gate: gate}
	rec := &linkRecorder{stored: true}
	v := NewSourceVerifier(rec, fetcher)
	ctx, cancel := context.WithCancel(context.Background())
	v.Start(ctx)
	v.Enqueue("p1", "F01", "https://a.org/ok", checkQuote)
	waitUntil(t, func() bool { return fetcher.running.Load() == 1 })
	cancel()
	waitUntil(t, func() bool { return fetcher.running.Load() == 0 })
	if len(rec.saved) != 0 {
		t.Fatalf("a download cut off by shutdown must not write a result, wrote %+v", rec.saved)
	}
}

// waitUntil polls cond until it holds or two seconds pass.
func waitUntil(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("condition not reached in time")
}
