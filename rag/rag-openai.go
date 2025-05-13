package rag

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type OpenAiLLM struct {
	Opts     LLMOpts
	Query    string
	Response string
}

// NewGeminiLLM is used to initalize a LLM that communicates with OpenAI's ChatGPT API.
func NewOpenAiLLM(opts LLMOpts) LLM {
	return &OpenAiLLM{
		Opts: opts,
	}
}

const openaiurl = "https://api.openai.com/v1/chat/completions"
const openairole = "user"

func (llm *OpenAiLLM) GenerateQuery(que string) (string, error) {
	payload := map[string]interface{}{
		"model": llm.Opts.Model,
		"messages": []map[string]string{
			{"role": openairole, "content": que},
		},
		// "context": llm.Opts.context, //Mapper Schema
		// "temperature": llm.Opts.Temp,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return llm.Query, fmt.Errorf("error marshaling json: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, openaiurl, bytes.NewBuffer(jsonData))
	if err != nil {
		return llm.Query, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", llm.Opts.ApiKey))
	req.Header.Set("OpenAI-Organization", llm.Opts.OrgId)
	req.Header.Set("OpenAI-Project", llm.Opts.ProjectId)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return llm.Query, fmt.Errorf("error sending request: %v", err)

	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return llm.Query, fmt.Errorf("non-ok http status: %v", resp.Status)

	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return llm.Query, fmt.Errorf("error reading response body: %v", err)

	}

	fmt.Printf("Response: %s\n", body)

	return llm.Query, nil
}

func (llm *OpenAiLLM) GenerateResponse(data interface{}, que string) (string, error) {

	return llm.Response, nil
}
