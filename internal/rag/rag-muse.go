package rag

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	defaultMuseBaseURL   = "https://api.meta.ai/v1/responses"
	defaultMuseModel     = "muse-wonder-1.0"
	defaultMuseTemp      = 0.2
	defaultMuseMaxTokens = 8192
)

const (
	sqlGenerationSchema = `{
		"type": "object",
		"properties": {
			"sql": {
				"type": "string",
				"description": "The valid read-only SQL SELECT query without markdown code fences."
			}
		}
	}`

	summaryGenerationSchema = `{
		"type": "object",
		"properties": {
			"summary": {
				"type": "string",
				"description": "Conversational natural language summary answering the user's question based on the queried database results."
			}
		}
	}`
)

// MuseLLM communicates with the Muse / Meta Model Responses API (/v1/responses).
type MuseLLM struct {
	Opts       LLMOpts
	Query      string
	Response   string
	httpClient *http.Client
}

// museResponsesEnvelope is the subset of a Responses API reply we read:
// completion status plus assistant message text.
type museResponsesEnvelope struct {
	Status string `json:"status"`
	Output []struct {
		Type    string `json:"type"`
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	} `json:"output"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error,omitempty"`
}

func (e museResponsesEnvelope) text() (json.RawMessage, error) {
	if e.Error != nil && e.Error.Message != "" {
		return nil, fmt.Errorf("muse API error: %s", e.Error.Message)
	}
	if e.Status != "" && e.Status != "completed" {
		return nil, fmt.Errorf("muse: response status %q", e.Status)
	}
	var b strings.Builder
	for _, item := range e.Output {
		if item.Type != "message" {
			continue
		}
		for _, c := range item.Content {
			if c.Type == "output_text" {
				b.WriteString(c.Text)
			}
		}
	}
	if b.Len() == 0 {
		return nil, fmt.Errorf("muse: no output_text in response")
	}
	return json.RawMessage(b.String()), nil
}

// enforceStrict makes a decoded JSON schema satisfy strict-mode rules:
// every object gets additionalProperties:false and all properties required.
func enforceStrict(node any) {
	n, ok := node.(map[string]any)
	if !ok {
		return
	}
	if props, ok := n["properties"].(map[string]any); ok {
		n["additionalProperties"] = false
		req := make([]string, 0, len(props))
		for k, p := range props {
			req = append(req, k)
			enforceStrict(p)
		}
		sort.Strings(req)
		n["required"] = req
	} else if t, _ := n["type"].(string); t == "object" {
		n["additionalProperties"] = false
	}
	if items, ok := n["items"]; ok {
		enforceStrict(items)
	}
	for _, key := range []string{"anyOf", "oneOf", "allOf"} {
		if arr, ok := n[key].([]any); ok {
			for _, s := range arr {
				enforceStrict(s)
			}
		}
	}
	for _, key := range []string{"$defs", "definitions"} {
		if defs, ok := n[key].(map[string]any); ok {
			for _, d := range defs {
				enforceStrict(d)
			}
		}
	}
}

// NewMuseLLM initializes an LLM client that communicates with the Muse Responses API.
func NewMuseLLM(opts LLMOpts) LLM {
	return &MuseLLM{
		Opts: opts,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
			Transport: &http.Transport{
				Proxy:                 http.ProxyFromEnvironment,
				DialContext:           (&net.Dialer{Timeout: 10 * time.Second}).DialContext,
				TLSHandshakeTimeout:   10 * time.Second,
				ResponseHeaderTimeout: 55 * time.Second,
			},
		},
	}
}

func (llm *MuseLLM) resolveEndpoint() string {
	base := strings.TrimSpace(llm.Opts.BaseUrl)
	if base == "" {
		return defaultMuseBaseURL
	}
	base = strings.TrimRight(base, "/")
	if strings.HasSuffix(base, "/responses") {
		return base
	}
	if strings.HasSuffix(base, "/v1") {
		return base + "/responses"
	}
	return base + "/v1/responses"
}

func (llm *MuseLLM) resolveModel() string {
	model := strings.TrimSpace(llm.Opts.Model)
	if model == "" {
		return defaultMuseModel
	}
	return model
}

func (llm *MuseLLM) resolveTemperature() float64 {
	if tempStr := strings.TrimSpace(llm.Opts.Temp); tempStr != "" {
		if tempVal, err := strconv.ParseFloat(tempStr, 64); err == nil {
			return tempVal
		}
	}
	return defaultMuseTemp
}

// CompleteRaw runs one stateless Responses call with server-enforced JSON schema output.
func (llm *MuseLLM) CompleteRaw(ctx context.Context, prompt, schemaName, schema string) (json.RawMessage, error) {
	var schemaDoc any
	if err := json.Unmarshal([]byte(schema), &schemaDoc); err != nil {
		return nil, fmt.Errorf("muse: bad schema: %w", err)
	}
	enforceStrict(schemaDoc)

	if schemaName == "" {
		schemaName = "NLPToSQLResult"
	}

	body, err := json.Marshal(map[string]any{
		"model":             llm.resolveModel(),
		"input":             prompt,
		"temperature":       llm.resolveTemperature(),
		"max_output_tokens": defaultMuseMaxTokens,
		"store":             false,
		"text": map[string]any{
			"format": map[string]any{
				"type":   "json_schema",
				"name":   schemaName,
				"schema": schemaDoc,
				"strict": true,
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("muse: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, llm.resolveEndpoint(), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if llm.Opts.ApiKey != "" {
		req.Header.Set("Authorization", "Bearer "+llm.Opts.ApiKey)
	}

	client := llm.httpClient
	if client == nil {
		client = &http.Client{Timeout: 60 * time.Second}
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		snip, _ := io.ReadAll(io.LimitReader(resp.Body, 500))
		return nil, fmt.Errorf("muse: status %s: %s", resp.Status, strings.TrimSpace(string(snip)))
	}

	var envelope museResponsesEnvelope
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return nil, err
	}
	return envelope.text()
}

// GenerateQuery generates a SQL SELECT query from natural language using the Muse Responses API.
func (llm *MuseLLM) GenerateQuery(que string) (string, error) {
	prompt := strings.Join([]string{
		queryInstructions(llm.Opts.Context),
		"",
		"User Question:",
		que,
	}, "\n")

	raw, err := llm.CompleteRaw(context.Background(), prompt, "SQLQueryResult", sqlGenerationSchema)
	if err != nil {
		return "", err
	}

	var parsed struct {
		SQL string `json:"sql"`
	}
	if err := json.Unmarshal(raw, &parsed); err == nil && strings.TrimSpace(parsed.SQL) != "" {
		llm.Query = strings.TrimSpace(parsed.SQL)
		return llm.Query, nil
	}

	// Fallback if raw output is already plain text SQL
	llm.Query = strings.TrimSpace(string(raw))
	return llm.Query, nil
}

// GenerateResponse summarizes the queried database rows into a natural language response using the Muse Responses API.
func (llm *MuseLLM) GenerateResponse(data interface{}, que string) (string, error) {
	prompt := strings.Join([]string{
		"Generate a summary from the next stream of input or text",
		fmt.Sprintf("Use these data: %v retrieved from the database in a conversational manner", data),
		fmt.Sprintf("Use this as context for the data returned: %v", que),
		"",
		"What is the summary of the data?",
	}, "\n")

	raw, err := llm.CompleteRaw(context.Background(), prompt, "DataSummaryResult", summaryGenerationSchema)
	if err != nil {
		return "", err
	}

	var parsed struct {
		Summary string `json:"summary"`
	}
	if err := json.Unmarshal(raw, &parsed); err == nil && strings.TrimSpace(parsed.Summary) != "" {
		llm.Response = strings.TrimSpace(parsed.Summary)
		return llm.Response, nil
	}

	llm.Response = strings.TrimSpace(string(raw))
	return llm.Response, nil
}

// GetModelContext returns nil, nil so callers use standard GenerateQuery for Muse.
func (llm *MuseLLM) GetModelContext() (interface{}, error) {
	return nil, nil
}
