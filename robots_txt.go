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
	"regexp"
	"strings"
	"time"

	"github.com/patrickmn/go-cache"
)

// Config the plugin configuration.
type Config struct {
	CustomRules  string `json:"customRules,omitempty"`
	Overwrite    bool   `json:"overwrite,omitempty"`
	AiRobotsTxt  bool   `json:"aiRobotsTxt,omitempty"`
	LastModified bool   `json:"lastModified,omitempty"`
	CacheTTL     int    `json:"cacheTTL,omitempty"`
	Block        bool   `json:"block,omitempty"`
}

// CreateConfig creates the default plugin configuration.
func CreateConfig() *Config {
	return &Config{
		CustomRules:  "",
		Overwrite:    false,
		AiRobotsTxt:  false,
		LastModified: false,
		CacheTTL:     30,
		Block:        false,
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
	customRules  string
	overwrite    bool
	aiRobotsTxt  bool
	lastModified bool
	cacheTTL     int
	block        bool
	next         http.Handler
}

var (
	c        *cache.Cache
	agentReg = regexp.MustCompile("^User-agent: (.+)$")
)

func getCachedAI() (string, error) {
	foo, found := c.Get("aiContent")
	if found {
		return foo.(string), nil
	}
	aiRobotsTxt, err := fetchAiRobotsTxt()
	if err != nil {
		log.Printf("unable to fetch ai.robots.txt: %v", err)
		return "", err
	}
	c.Set("aiContent", aiRobotsTxt, cache.DefaultExpiration)
	return aiRobotsTxt, nil
}

func GetRegex() (*regexp.Regexp, error) {
	foo, found := c.Get("reg")
	if found {
		return foo.(*regexp.Regexp), nil
	}
	// TODO
	aiResp, aiErr := getCachedAI()
	if aiErr != nil {
		return nil, aiErr
	}

	quotedBotPatterns := []string{}

	for _, line := range strings.Split(strings.TrimSuffix(aiResp, "\n"), "\n") {
		match := agentReg.FindStringSubmatch(line)
		if match != nil {
			quotedBotPatterns = append(quotedBotPatterns, regexp.QuoteMeta(match[1]))
		}
	}
	if len(quotedBotPatterns) == 0 {
		log.Printf("No matched User-Agents from ai.robots.txt ?")
		return nil, nil
	}
	matcherCode := fmt.Sprintf("(?i)(%s)", strings.Join(quotedBotPatterns, "|"))
	matcher, err := regexp.Compile(matcherCode)
	if err != nil {
		log.Printf("unable to compile regex: %v", err)
		return nil, err
	}
	c.Set("reg", matcher, cache.DefaultExpiration)
	return matcher, nil
}

func BlockAgent(res *http.ResponseWriter) {
	(*res).Header().Set("Content-Type", "text/plain; charset=utf-8")
	(*res).WriteHeader(http.StatusForbidden)
	_, _ = (*res).Write([]byte("Access denied"))
}

// New created a new Demo plugin.
func New(ctx context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	if len(config.CustomRules) == 0 && !config.AiRobotsTxt {
		return nil, fmt.Errorf("set customRules or set aiRobotsTxt to true")
	}

	c = cache.New(time.Duration(config.CacheTTL)*time.Minute, 10*time.Minute)

	return &RobotsTxtPlugin{
		customRules:  config.CustomRules,
		overwrite:    config.Overwrite,
		aiRobotsTxt:  config.AiRobotsTxt,
		lastModified: config.LastModified,
		cacheTTL:     config.CacheTTL,
		block:        config.Block,
		next:         next,
	}, nil
}

func (p *RobotsTxtPlugin) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	if strings.ToLower(req.URL.Path) != "/robots.txt" {

		if p.block && p.aiRobotsTxt {
			for _, uaHeader := range req.Header.Values("User-Agent") {
				agentMatch, err := GetRegex()
				if err == nil {
					if agentMatch != nil && agentMatch.MatchString(uaHeader) {
						BlockAgent(&rw)
						return
					}
				} else {
					log.Printf("unable to match against User-Agent: %v", err)
				}
			}
		}

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

	if !p.overwrite && wrappedWriter.backendStatusCode != http.StatusNotFound {
		body = wrappedWriter.buffer.String() + "\n"
	}

	body += "# The following content was added on the fly by the Robots.txt Traefik plugin: " +
		"https://plugins.traefik.io/plugins/681b2f3fba3486128fc34fae/robots-txt-plugin\n"

	if p.aiRobotsTxt {
		aiRobotsTxt, err := getCachedAI()
		if err != nil {
			log.Printf("unable to fetch ai.robots.txt: %v", err)
		}
		body += aiRobotsTxt
	}
	body += p.customRules

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

func fetchAiRobotsTxt() (string, error) {
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
