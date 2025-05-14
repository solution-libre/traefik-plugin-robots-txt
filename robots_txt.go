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

// Package traefik_plugin_robots_txt a plugin to complete robots.txt file.
package traefik_plugin_robots_txt

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
)

// Config the plugin configuration.
type Config struct {
	AdditionalRules string `json:"additionalRules,omitempty"`
	OverwriteRules  string `json:"overwriteRules,omitempty"`
	AiRobotsTxt     bool   `json:"aiRobotsTxt,omitempty"`
	LastModified    bool   `json:"lastModified,omitempty"`
}

// CreateConfig creates the default plugin configuration.
func CreateConfig() *Config {
	return &Config{
		AdditionalRules: "",
		OverwriteRules:  "",
		AiRobotsTxt:     false,
		LastModified:    false,
	}
}

type responseWriter struct {
	buffer       bytes.Buffer
	lastModified bool
	wroteHeader  bool

	http.ResponseWriter
	backendStatusCode int
	statusCode        int
}

// RobotsTxtPlugin a robots.txt plugin.
type RobotsTxtPlugin struct {
	additionalRules string
	overwriteRules  string
	aiRobotsTxt     bool
	lastModified    bool
	next            http.Handler
}

// New created a new Demo plugin.
func New(ctx context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	if len(config.AdditionalRules) == 0 && !config.AiRobotsTxt && len(config.OverwriteRules) == 0 {
		return nil, fmt.Errorf("set additionalRules, overwriteRules, or set ai.robot.txt to true")
	}

	return &RobotsTxtPlugin{
		additionalRules: config.AdditionalRules,
		overwriteRules:  config.OverwriteRules,
		aiRobotsTxt:     config.AiRobotsTxt,
		lastModified:    config.LastModified,
		next:            next,
	}, nil
}

func (p *RobotsTxtPlugin) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	if !strings.HasSuffix(strings.ToLower(req.URL.Path), "/robots.txt") {
		p.next.ServeHTTP(rw, req)
		return
	}

	wrappedWriter := &responseWriter{
		lastModified:      p.lastModified,
		ResponseWriter:    rw,
		backendStatusCode: http.StatusOK,
		statusCode:        http.StatusOK,
	}
	p.next.ServeHTTP(wrappedWriter, req)

	if wrappedWriter.backendStatusCode == http.StatusNotModified {
		return
	}

	var body string

	//if OverwriteRules is set, use it and skip everything else
	if p.overwriteRules != "" {
		body = p.overwriteRules
	} else {
		if wrappedWriter.backendStatusCode != http.StatusNotFound {
			body = wrappedWriter.buffer.String()
		}

		if p.aiRobotsTxt {
			aiRobotsTxt, err := p.fetchAiRobotsTxt()
			if err != nil {
				log.Printf("unable to fetch ai.robots.txt: %v", err)
			}
			body += aiRobotsTxt
		}

		body += p.additionalRules
	}

	_, err := rw.Write([]byte(body))
	if err != nil {
		log.Printf("unable to write body: %v", err)
	}
}

func (r *responseWriter) WriteHeader(statusCode int) {
	if !r.lastModified {
		r.ResponseWriter.Header().Del("Last-Modified")
	}

	r.wroteHeader = true
	r.backendStatusCode = statusCode
	if statusCode != http.StatusNotFound {
		r.statusCode = statusCode
	} else {
		r.statusCode = http.StatusOK
	}

	r.ResponseWriter.Header().Set("Content-Type", "text/plain")

	// Delegates the Content-Length Header creation to the final body write.
	r.ResponseWriter.Header().Del("Content-Length")

	r.ResponseWriter.WriteHeader(r.statusCode)
}

func (r *responseWriter) Write(p []byte) (int, error) {
	if !r.wroteHeader {
		r.WriteHeader(http.StatusOK)
	}

	return r.buffer.Write(p)
}

func (r *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := r.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("%T is not a http.Hijacker", r.ResponseWriter)
	}

	return hijacker.Hijack()
}

func (r *responseWriter) Flush() {
	if flusher, ok := r.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (p *RobotsTxtPlugin) fetchAiRobotsTxt() (string, error) {
	backendURL := "https://raw.githubusercontent.com/ai-robots-txt/ai.robots.txt/refs/heads/main/robots.txt"

	resp, err := http.Get(backendURL)
	if err != nil {
		return "", err
	}

	defer func() {
		if err = resp.Body.Close(); err != nil {
			log.Printf("Error closing HTTP response: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP status code is not 200")
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(content), nil
}
