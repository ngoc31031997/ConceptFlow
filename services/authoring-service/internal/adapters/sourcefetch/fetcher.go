package sourcefetch

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"net/netip"
	"regexp"
	"strings"
	"syscall"
	"time"

	"golang.org/x/text/encoding/htmlindex"

	"authoring/internal/application"
	"authoring/internal/domain"
)

const (
	// maxRedirects is how many redirects one fetch follows.
	maxRedirects = 5
	// maxConcurrentFetches is how many pages the service downloads at once.
	maxConcurrentFetches = 4
	// maxPageBytes is the largest page read.
	maxPageBytes = 5 << 20
	dialTimeout  = 4 * time.Second
	totalTimeout = 12 * time.Second
	userAgent    = "ConceptFlow-research-check/1.0"
)

var metaCharset = regexp.MustCompile(`(?i)<meta[^>]+charset=["']?([a-zA-Z0-9_-]+)`)

// Fetcher downloads a page as text for the source check. It sends no cookies
// or credentials, runs no scripts, and reaches only public hosts: names and
// addresses are checked before connecting and the address actually connected
// is checked again, so a name that changes its answer cannot lead inside.
// Implements application.SourceFetcher.
type Fetcher struct {
	client    *http.Client
	sem       chan struct{}
	allowAddr func(netip.Addr) bool
	allowPort func(string) bool
}

// New builds the fetcher the service uses.
func New() *Fetcher {
	return newFetcher(publicAddress, defaultPort, totalTimeout)
}

func newFetcher(allowAddr func(netip.Addr) bool, allowPort func(string) bool, timeout time.Duration) *Fetcher {
	f := &Fetcher{
		sem:       make(chan struct{}, maxConcurrentFetches),
		allowAddr: allowAddr,
		allowPort: allowPort,
	}
	dialer := &net.Dialer{
		Timeout: dialTimeout,
		// Control sees the address about to be connected, after name resolution.
		Control: func(_, address string, _ syscall.RawConn) error {
			host, _, err := net.SplitHostPort(address)
			if err != nil {
				return &application.FetchError{Reason: reasonInternal}
			}
			ip, err := netip.ParseAddr(host)
			if err != nil || !allowAddr(ip) {
				return &application.FetchError{Reason: reasonInternal}
			}
			return nil
		},
	}
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(address)
			if err != nil {
				return nil, err
			}
			ips, err := net.DefaultResolver.LookupNetIP(ctx, "ip", host)
			if err != nil {
				return nil, err
			}
			// Every address the name resolves to must be allowed, not just the one used.
			for _, ip := range ips {
				if !allowAddr(ip) {
					return nil, &application.FetchError{Reason: reasonInternal}
				}
			}
			var lastErr error
			for _, ip := range ips {
				conn, err := dialer.DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
				if err == nil {
					return conn, nil
				}
				lastErr = err
			}
			return nil, lastErr
		},
		DisableKeepAlives:     true,
		ResponseHeaderTimeout: timeout,
	}
	f.client = &http.Client{
		Transport: transport,
		Timeout:   timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) > maxRedirects {
				return &application.FetchError{Reason: reasonNoRedirect}
			}
			_, err := checkURL(req.URL.String(), allowAddr, allowPort)
			return err
		},
	}
	return f
}

// Fetch downloads rawURL and returns its text. See application.SourceFetcher.
func (f *Fetcher) Fetch(ctx context.Context, rawURL string) (application.SourcePage, error) {
	u, err := checkURL(rawURL, f.allowAddr, f.allowPort)
	if err != nil {
		return application.SourcePage{}, err
	}
	select {
	case f.sem <- struct{}{}:
		defer func() { <-f.sem }()
	case <-ctx.Done():
		return application.SourcePage{}, ctx.Err()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return application.SourcePage{}, &application.FetchError{Reason: reasonNotWeb}
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,text/plain;q=0.9")
	resp, err := f.client.Do(req)
	if err != nil {
		return application.SourcePage{}, f.fetchFailure(ctx, err)
	}
	defer resp.Body.Close()
	if reason := statusReason(resp.StatusCode); reason != "" {
		return application.SourcePage{}, &application.FetchError{Reason: reason}
	}
	mediaType, params, _ := mime.ParseMediaType(resp.Header.Get("Content-Type"))
	if !readableType(mediaType) {
		return application.SourcePage{}, &application.FetchError{Reason: fmt.Sprintf("loại tệp chưa đọc được (%s)", orUnknown(mediaType))}
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxPageBytes+1))
	if err != nil {
		return application.SourcePage{}, f.fetchFailure(ctx, err)
	}
	if len(body) > maxPageBytes {
		return application.SourcePage{}, &application.FetchError{Reason: "trang quá lớn (hơn 5 MB)"}
	}
	text := decodeBody(body, params["charset"])
	if mediaType != "text/plain" {
		text = domain.StripHTML(text)
	}
	return application.SourcePage{Text: text}, nil
}

// fetchFailure turns a transport error into the reason the Creator reads. A
// cancelled caller is returned as the context's own error.
func (f *Fetcher) fetchFailure(ctx context.Context, err error) error {
	if ctx.Err() != nil && !errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return ctx.Err()
	}
	var fe *application.FetchError
	if errors.As(err, &fe) {
		return fe
	}
	var ne net.Error
	if errors.Is(err, context.DeadlineExceeded) || (errors.As(err, &ne) && ne.Timeout()) {
		return &application.FetchError{Reason: "quá thời gian chờ trang"}
	}
	return &application.FetchError{Reason: "không nối được tới trang"}
}

// statusReason explains an HTTP status that is not a readable page, or "" for a 2xx.
func statusReason(code int) string {
	switch {
	case code >= 200 && code < 300:
		return ""
	case code == http.StatusUnauthorized, code == http.StatusPaymentRequired, code == http.StatusForbidden:
		return fmt.Sprintf("trang trả phí hoặc cần đăng nhập (HTTP %d)", code)
	case code == http.StatusNotFound:
		return "không tìm thấy trang (404)"
	default:
		return fmt.Sprintf("trang từ chối hoặc lỗi (HTTP %d)", code)
	}
}

func readableType(mediaType string) bool {
	switch mediaType {
	case "text/html", "application/xhtml+xml", "text/plain":
		return true
	}
	return false
}

func orUnknown(s string) string {
	if s == "" {
		return "không rõ"
	}
	return s
}

// decodeBody reads body as text in its declared character set (the header's,
// else the page's own <meta>), as UTF-8 when none is declared or known.
func decodeBody(body []byte, headerCharset string) string {
	name := headerCharset
	if name == "" {
		head := body
		if len(head) > 2048 {
			head = head[:2048]
		}
		if m := metaCharset.FindSubmatch(head); m != nil {
			name = string(m[1])
		}
	}
	if name != "" {
		if enc, err := htmlindex.Get(strings.TrimSpace(name)); err == nil {
			if decoded, err := enc.NewDecoder().Bytes(body); err == nil {
				body = decoded
			}
		}
	}
	return strings.ToValidUTF8(string(body), "�")
}
