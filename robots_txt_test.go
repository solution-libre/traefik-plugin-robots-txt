// Copyright 2020 Containous SAS
// Copyright 2020 Traefik Labs
// Copyright 2025 Solution Libre
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package traefik_plugin_robots_txt_test

import (
	"bytes"
	"context"
	"errors"
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

	if recorder.Code != http.StatusOK {
		t.Errorf("got status code %d, want %d", http.StatusOK, recorder.Code)
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

	if recorder.Code != http.StatusOK {
		t.Errorf("got status code %d, want %d", http.StatusOK, recorder.Code)
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
		t.Fatal(errors.New("an error should raised up"))
	} else {
		errMsg := "set additionnal rules or set ai.robot.txt to true"
		if err.Error() != errMsg {
			t.Errorf("got err message %s, want %s", err.Error(), errMsg)
		}
	}
}
