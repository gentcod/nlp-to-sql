package rag

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type OpenAiLLM struct {
	Opts     LLMOpts
	Query    string
	Response string
}

// NewOpenAiLLM initializes an LLM that communicates with OpenAI's ChatGPT API.
func NewOpenAiLLM(opts LLMOpts) LLM {
	return &OpenAiLLM{
		Opts: opts,
	}
}

const openaiurl = "https://api.openai.com/v1/chat/completions"

func (llm *OpenAiLLM) completeChat(systemPrompt, userPrompt string) (string, error) {
	modelName := strings.TrimSpace(llm.Opts.Model)
	if modelName == "" {
		modelName = "gpt-4o-mini"
	}

	payload := map[string]interface{}{
		"model": modelName,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": userPrompt},
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("error marshaling json: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, openaiurl, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", llm.Opts.ApiKey))
	if llm.Opts.OrgId != "" {
		req.Header.Set("OpenAI-Organization", llm.Opts.OrgId)
	}
	if llm.Opts.ProjectId != "" {
		req.Header.Set("OpenAI-Project", llm.Opts.ProjectId)
	}

	client := &http.Client{Timeout: 45 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response body: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("non-ok http status %v: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", fmt.Errorf("error decoding openai response: %v", err)
	}
	if len(parsed.Choices) == 0 {
		return "", fmt.Errorf("openai returned no choices")
	}

	return strings.TrimSpace(parsed.Choices[0].Message.Content), nil
}

func (llm *OpenAiLLM) GenerateQuery(que string) (string, error) {
	query, err := llm.completeChat(queryInstructions(llm.Opts.Context), que)
	if err != nil {
		return "", err
	}
	llm.Query = query
	return llm.Query, nil
}

func (llm *OpenAiLLM) GenerateResponse(data interface{}, que string) (string, error) {
	sysPrompt := strings.Join([]string{
		"Generate a summary from the next stream of input or text",
		fmt.Sprintf("Use these data: %v retrieved from the database in a conversational manner", data),
		fmt.Sprintf("Use this as context for the data returned: %v", que),
	}, "\n")

	summary, err := llm.completeChat(sysPrompt, "What is the summary of the data?")
	if err != nil {
		return "", err
	}
	llm.Response = summary
	return llm.Response, nil
}

func (llm *OpenAiLLM) GetModelContext() (interface{}, error) {
	return nil, nil
}
