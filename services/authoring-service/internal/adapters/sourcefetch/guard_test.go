package sourcefetch

import (
	"errors"
	"net/netip"
	"strings"
	"testing"

	"authoring/internal/application"
)

func TestCheckURLRefusesInternalAndOddLinks(t *testing.T) {
	refused := []string{
		"http://localhost/",
		"http://LOCALHOST:80/x",
		"http://127.0.0.1/",
		"http://127.1.2.3/",
		"http://10.0.0.5/",
		"http://172.16.0.1/",
		"http://172.31.255.255/",
		"http://192.168.1.1/",
		"http://169.254.169.254/latest/meta-data/",
		"http://100.64.0.1/",
		"http://0.0.0.0/",
		"http://[::1]/",
		"http://[fd00::1]/",
		"http://[fe80::1]/",
		"http://[::ffff:10.0.0.1]/",
		"http://[::ffff:169.254.169.254]/",
		"http://rabbitmq:5672/",
		"http://postgres/",
		"http://authoring-service/v1/projects",
		"http://2130706433/",
		"http://printer.local/",
		"https://db.internal/",
		"https://nas.lan/",
		"ftp://example.com/",
		"file:///etc/passwd",
		"javascript:alert(1)",
		"https://example.com:8443/",
		"https://example.com:22/",
		"https://user:pw@example.com/",
		"",
		"https://",
	}
	for _, raw := range refused {
		t.Run(raw, func(t *testing.T) {
			_, err := checkURL(raw, publicAddress, defaultPort)
			var fe *application.FetchError
			if !errors.As(err, &fe) {
				t.Fatalf("checkURL(%q) err = %v, want a FetchError", raw, err)
			}
		})
	}
}

func TestCheckURLAcceptsPublicLinks(t *testing.T) {
	for _, raw := range []string{
		"https://example.com/",
		"http://en.wikipedia.org/wiki/Fig_wasp#History",
		"https://example.com:443/page?q=1",
		"http://8.8.8.8/",
		"https://[2606:4700:4700::1111]/",
	} {
		u, err := checkURL(raw, publicAddress, defaultPort)
		if err != nil {
			t.Errorf("checkURL(%q) = %v, want accepted", raw, err)
			continue
		}
		if u.Fragment != "" {
			t.Errorf("checkURL(%q) kept the fragment %q", raw, u.Fragment)
		}
	}
}

func TestCheckURLReasonNamesTheProblem(t *testing.T) {
	_, err := checkURL("http://169.254.169.254/", publicAddress, defaultPort)
	if err == nil || !strings.Contains(err.Error(), "nội bộ") {
		t.Fatalf("err = %v, want the internal-address reason", err)
	}
	_, err = checkURL("https://example.com:8443/", publicAddress, defaultPort)
	if err == nil || !strings.Contains(err.Error(), "cổng") {
		t.Fatalf("err = %v, want the port reason", err)
	}
}

func TestPublicAddress(t *testing.T) {
	for addr, want := range map[string]bool{
		"8.8.8.8": true, "1.1.1.1": true, "2606:4700:4700::1111": true,
		"127.0.0.1": false, "10.1.2.3": false, "172.20.0.2": false, "192.168.0.9": false,
		"169.254.1.1": false, "224.0.0.1": false, "0.1.2.3": false, "100.100.0.1": false,
		"198.18.0.5": false, "240.0.0.1": false, "255.255.255.255": false,
		"::1": false, "fc00::1": false, "fe80::1": false, "::ffff:127.0.0.1": false,
	} {
		if got := publicAddress(netip.MustParseAddr(addr)); got != want {
			t.Errorf("publicAddress(%s) = %v, want %v", addr, got, want)
		}
	}
}
