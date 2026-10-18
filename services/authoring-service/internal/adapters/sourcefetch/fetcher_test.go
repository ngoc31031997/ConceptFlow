package sourcefetch

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"authoring/internal/application"
)

// testFetcher reaches only loopback, so a httptest server can stand in for a
// site. Every other address stays refused by the same rules as production.
func testFetcher(timeout time.Duration) *Fetcher {
	return newFetcher(
		func(ip netip.Addr) bool { return ip.IsLoopback() },
		func(string) bool { return true },
		timeout,
	)
}

func fetchReason(t *testing.T, err error) string {
	t.Helper()
	var fe *application.FetchError
	if !errors.As(err, &fe) {
		t.Fatalf("err = %v, want a FetchError", err)
	}
	return fe.Reason
}

func TestFetchReturnsPageTextWithoutMarkup(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("User-Agent"); !strings.HasPrefix(got, "ConceptFlow") {
			t.Errorf("User-Agent = %q, want the app's name", got)
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, `<html><script>bad()</script><p>Quả sung là một chùm hoa &amp; kín.</p></html>`)
	}))
	defer srv.Close()
	page, err := testFetcher(2*time.Second).Fetch(context.Background(), srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(page.Text, "Quả sung là một chùm hoa & kín.") || strings.Contains(page.Text, "bad()") {
		t.Fatalf("Text = %q, want visible text only", page.Text)
	}
}

func TestFetchDecodesDeclaredCharset(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		// "café" in ISO-8859-1, declared in the page's own <meta>.
		_, _ = w.Write([]byte("<meta charset=\"iso-8859-1\"><p>caf\xe9</p>"))
	}))
	defer srv.Close()
	page, err := testFetcher(2*time.Second).Fetch(context.Background(), srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(page.Text, "café") {
		t.Fatalf("Text = %q, want café", page.Text)
	}
}

func TestFetchExplainsRefusals(t *testing.T) {
	cases := map[int]string{403: "trả phí", 402: "trả phí", 404: "404", 429: "từ chối", 503: "từ chối"}
	for code, want := range cases {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(code)
			}))
			defer srv.Close()
			_, err := testFetcher(2*time.Second).Fetch(context.Background(), srv.URL)
			if reason := fetchReason(t, err); !strings.Contains(reason, want) {
				t.Fatalf("reason = %q, want it to mention %q", reason, want)
			}
		})
	}
}

func TestFetchFollowsFiveRedirectsButNotSix(t *testing.T) {
	mux := http.NewServeMux()
	for i := 0; i < 5; i++ {
		i := i
		mux.HandleFunc(fmt.Sprintf("/r%d", i), func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, fmt.Sprintf("/r%d", i+1), http.StatusFound)
		})
	}
	mux.HandleFunc("/r5", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprint(w, "arrived")
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	f := testFetcher(2 * time.Second)
	page, err := f.Fetch(context.Background(), srv.URL+"/r0")
	if err != nil || page.Text != "arrived" {
		t.Fatalf("five redirects: text=%q err=%v, want arrived", page.Text, err)
	}
	for i := 0; i <= 6; i++ {
		i := i
		mux.HandleFunc(fmt.Sprintf("/s%d", i), func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, fmt.Sprintf("/s%d", i+1), http.StatusFound)
		})
	}
	_, err = f.Fetch(context.Background(), srv.URL+"/s0")
	if reason := fetchReason(t, err); !strings.Contains(reason, "chuyển hướng") {
		t.Fatalf("six redirects reason = %q, want the redirect limit", reason)
	}
}

func TestFetchRefusesRedirectToInternalAddress(t *testing.T) {
	for _, target := range []string{"http://169.254.169.254/latest/meta-data/", "http://10.0.0.1/", "http://rabbitmq:15672/"} {
		t.Run(target, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Redirect(w, r, target, http.StatusFound)
			}))
			defer srv.Close()
			_, err := testFetcher(2*time.Second).Fetch(context.Background(), srv.URL)
			if reason := fetchReason(t, err); !strings.Contains(reason, "nội bộ") {
				t.Fatalf("reason = %q, want the internal-address reason", reason)
			}
		})
	}
}

func TestFetchRefusesLoopbackWithTheRealRules(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("the production rules must not reach a loopback server")
	}))
	defer srv.Close()
	_, err := New().Fetch(context.Background(), srv.URL)
	if reason := fetchReason(t, err); !strings.Contains(reason, "nội bộ") && !strings.Contains(reason, "cổng") {
		t.Fatalf("reason = %q, want the link refused", reason)
	}
}

func TestFetchRefusesUnreadableTypesAndHugePages(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/pdf":
			w.Header().Set("Content-Type", "application/pdf")
			fmt.Fprint(w, "%PDF-1.4")
		case "/big":
			w.Header().Set("Content-Type", "text/html")
			chunk := []byte(strings.Repeat("a", 1<<20))
			for i := 0; i < 6; i++ {
				_, _ = w.Write(chunk)
			}
		}
	}))
	defer srv.Close()
	f := testFetcher(5 * time.Second)
	_, err := f.Fetch(context.Background(), srv.URL+"/pdf")
	if reason := fetchReason(t, err); !strings.Contains(reason, "loại tệp") {
		t.Fatalf("pdf reason = %q", reason)
	}
	_, err = f.Fetch(context.Background(), srv.URL+"/big")
	if reason := fetchReason(t, err); !strings.Contains(reason, "quá lớn") {
		t.Fatalf("big reason = %q", reason)
	}
}

func TestFetchTimesOut(t *testing.T) {
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-release:
		case <-r.Context().Done():
		}
	}))
	defer srv.Close()
	defer close(release)
	_, err := testFetcher(200*time.Millisecond).Fetch(context.Background(), srv.URL)
	if reason := fetchReason(t, err); !strings.Contains(reason, "quá thời gian") {
		t.Fatalf("reason = %q, want the time-out reason", reason)
	}
}

func TestFetchReturnsCallerCancellation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer srv.Close()
	ctx, cancel := context.WithCancel(context.Background())
	go func() { time.Sleep(50 * time.Millisecond); cancel() }()
	_, err := testFetcher(5*time.Second).Fetch(ctx, srv.URL)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
}

func TestFetchSendsNoCookies(t *testing.T) {
	var secondHadCookie atomic.Bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		if r.URL.Path == "/first" {
			http.SetCookie(w, &http.Cookie{Name: "session", Value: "secret"})
			return
		}
		if r.Header.Get("Cookie") != "" || r.Header.Get("Authorization") != "" {
			secondHadCookie.Store(true)
		}
	}))
	defer srv.Close()
	f := testFetcher(2 * time.Second)
	for _, path := range []string{"/first", "/second"} {
		if _, err := f.Fetch(context.Background(), srv.URL+path); err != nil {
			t.Fatal(err)
		}
	}
	if secondHadCookie.Load() {
		t.Fatal("the second request carried a cookie or credentials")
	}
}

func TestFetcherLimitsConcurrentDownloads(t *testing.T) {
	if got := cap(New().sem); got != maxConcurrentFetches {
		t.Fatalf("concurrency = %d, want %d", got, maxConcurrentFetches)
	}
	var running, peak atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := running.Add(1)
		for {
			p := peak.Load()
			if n <= p || peak.CompareAndSwap(p, n) {
				break
			}
		}
		time.Sleep(60 * time.Millisecond)
		running.Add(-1)
		w.Header().Set("Content-Type", "text/plain")
	}))
	defer srv.Close()
	f := testFetcher(5 * time.Second)
	done := make(chan struct{})
	for i := 0; i < 10; i++ {
		go func() {
			_, _ = f.Fetch(context.Background(), srv.URL)
			done <- struct{}{}
		}()
	}
	for i := 0; i < 10; i++ {
		<-done
	}
	if peak.Load() > maxConcurrentFetches {
		t.Fatalf("peak concurrent downloads = %d, want at most %d", peak.Load(), maxConcurrentFetches)
	}
}
