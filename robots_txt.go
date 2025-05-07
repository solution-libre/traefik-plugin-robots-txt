// Package traefik_plugin_robots_txt a plugin to complete robots.txt file.
package traefik_plugin_robots_txt

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
)

// Config the plugin configuration.
type Config struct {
	AdditionalRules string `json:"additionalRules,omitempty"`
	AiRobotsTxt     bool   `json:"aiRobotsTxt,omitempty"`
}

// CreateConfig creates the default plugin configuration.
func CreateConfig() *Config {
	return &Config{
		AdditionalRules: "",
		AiRobotsTxt:     false,
	}
}

type responseWriter struct {
	buffer bytes.Buffer

	http.ResponseWriter
}

// RobotsTxtPlugin a robots.txt plugin.
type RobotsTxtPlugin struct {
	additionalRules string
	aiRobotsTxt     bool
	next            http.Handler
}

// New created a new Demo plugin.
func New(ctx context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	if len(config.AdditionalRules) == 0 && !config.AiRobotsTxt {
		return nil, fmt.Errorf("set additionnal rules or set ai.robot.txt to true")
	}

	return &RobotsTxtPlugin{
		additionalRules: config.AdditionalRules,
		aiRobotsTxt:     config.AiRobotsTxt,
		next:            next,
	}, nil
}

func (p *RobotsTxtPlugin) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	wrappedWriter := &responseWriter{
		ResponseWriter: rw,
	}

	if !strings.HasSuffix(req.URL.Path, "/robots.txt") {
		p.next.ServeHTTP(rw, req)
		return
	}

	p.next.ServeHTTP(wrappedWriter, req)

	body := wrappedWriter.buffer.String()

	if p.aiRobotsTxt {
		aiRobotsTxt, err := p.fetchAiRobotsTxt()
		if err != nil {
			log.Printf("unable to fetch ai.robots.txt: %v", err)
		}
		body += aiRobotsTxt
	}

	body += p.additionalRules

	rw.Header().Set("Content-Type", "text/plain")
	rw.WriteHeader(http.StatusOK)
	_, err := rw.Write([]byte(body))
	if err != nil {
		log.Printf("unable to write body: %v", err)
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
			fmt.Printf("Error closing HTTP response: %v", err)
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
