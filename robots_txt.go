// Package plugindemo a demo plugin.
package traefik_plugin_robots_txt

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strings"
)

// Config the plugin configuration.
type Config struct {
	AdditionalRules string `json:"headers,omitempty"`
	LastModified    bool   `json:"lastModified,omitempty"`
}

// CreateConfig creates the default plugin configuration.
func CreateConfig() *Config {
	return &Config{
		AdditionalRules: "",
	}
}

type responseWriter struct {
	buffer       bytes.Buffer
	lastModified bool
	wroteHeader  bool

	http.ResponseWriter
}

type RobotsTxtPlugin struct {
	additionalRules string
	lastModified    bool
	next            http.Handler
}

// New created a new Demo plugin.
func New(ctx context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	if len(config.AdditionalRules) == 0 {
		return nil, fmt.Errorf("additionnal rules cannot be empty")
	}

	return &RobotsTxtPlugin{
		additionalRules: config.AdditionalRules,
		next:            next,
	}, nil
}

func (p *RobotsTxtPlugin) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	wrappedWriter := &responseWriter{
		lastModified:   p.lastModified,
		ResponseWriter: rw,
	}

	if strings.HasSuffix(req.URL.Path, "/robots.txt") {

		p.next.ServeHTTP(wrappedWriter, req)

		body := wrappedWriter.buffer.String()

		completeContent := body + p.additionalRules

		rw.Header().Set("Content-Type", "text/plain")
		rw.WriteHeader(http.StatusOK)
		rw.Write([]byte(completeContent))
	} else {
		p.next.ServeHTTP(rw, req)
	}
}
