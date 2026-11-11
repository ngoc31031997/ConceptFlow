package rendering

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"visual/internal/application"
)

func serve(t *testing.T, status int, body string) *Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/fonts" || r.Method != http.MethodGet {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return NewClient(srv.URL, time.Second)
}

func TestFonts_ReadsTheList(t *testing.T) {
	c := serve(t, 200, `{"fonts":["Be Vietnam Pro","Montserrat","Cormorant Garamond"],"default":"Be Vietnam Pro"}`)
	fonts, err := c.Fonts(context.Background())
	if err != nil || len(fonts) != 3 || fonts[1] != "Montserrat" {
		t.Fatalf("%v %v", fonts, err)
	}
}

func TestFonts_ErrorsNameRendering(t *testing.T) {
	cases := map[string]*Client{
		"500":        serve(t, 500, `boom`),
		"bad json":   serve(t, 200, `not json`),
		"empty list": serve(t, 200, `{"fonts":[]}`),
		"wrong type": serve(t, 200, `{"fonts":"Be Vietnam Pro"}`),
	}
	srv := httptest.NewServer(http.NotFoundHandler())
	dead := NewClient(srv.URL, time.Second)
	srv.Close()
	cases["unreachable"] = dead
	for name, c := range cases {
		if _, err := c.Fonts(context.Background()); !errors.Is(err, application.ErrRenderingFailed) {
			t.Errorf("%s: %v", name, err)
		}
	}
}
