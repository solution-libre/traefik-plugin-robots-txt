package traefik_plugin_robots_txt_test

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	plugin "github.com/solution-libre/traefik-plugin-robots-txt"
)

func TestAdditionalRules(t *testing.T) {
	cfg := plugin.CreateConfig()
	cfg.AdditionalRules = "\nUser-agent: *\nDisallow: /private/\n"

	ctx := context.Background()
	next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {})

	handler, err := plugin.New(ctx, next, cfg, "robots-txt-plugin")
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

func TestAiRobotsTxt(t *testing.T) {
	cfg := plugin.CreateConfig()
	cfg.AiRobotsTxt = true

	ctx := context.Background()
	next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {})

	handler, err := plugin.New(ctx, next, cfg, "robots-txt-plugin")
	if err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost/robots.txt", nil)
	if err != nil {
		t.Fatal(err)
	}

	handler.ServeHTTP(recorder, req)

	if strings.HasSuffix(recorder.Body.String(), "Disallow: /") {
		t.Errorf("got body %s, want terminated by %s", recorder.Body.String(), "Disallow: /")
	}
}

func TestNoOption(t *testing.T) {
	cfg := plugin.CreateConfig()
	cfg.AdditionalRules = ""
	cfg.AiRobotsTxt = false

	ctx := context.Background()
	next := http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {})

	_, err := plugin.New(ctx, next, cfg, "robots-txt-plugin")
	if err == nil {
		t.Fatal(fmt.Errorf("an error should raised up"))
	} else {
		errMsg := "set additionnal rules or set ai.robot.txt to true"
		if err.Error() != errMsg {
			t.Errorf("got err message %s, want %s", err.Error(), errMsg)
		}
	}
}
