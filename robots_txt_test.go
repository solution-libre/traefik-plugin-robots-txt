package traefik_plugin_robots_txt_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	plugin "github.com/solution-libre/traefik-plugin-robots-txt"
)

func TestDemo(t *testing.T) {
	cfg := plugin.CreateConfig()
	cfg.AdditionalRules = "\nUser-agent: *\nDisallow: /private/\n"

	ctx := context.Background()
	next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {})

	handler, err := plugin.New(ctx, next, cfg, "demo-plugin")
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost/robots.txt", nil)
	if err != nil {
		t.Fatal(err)
	}

	handler.ServeHTTP(recorder, req)

	if !bytes.Equal([]byte("\nUser-agent: *\nDisallow: /private/\n"), recorder.Body.Bytes()) {
		t.Errorf("got body %q, want %q", recorder.Body.Bytes(), "\nUser-agent: *\nDisallow: /private/\n")
	}
}
