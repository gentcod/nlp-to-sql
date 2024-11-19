package rag

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// const testurl = "https://foodie-f019.onrender.com/api/v1/account/login"
// const llmurl = "https://api.openai.com/v1/models"
const llmurl = "https://api.openai.com/v1/chat/completions"
const role = "user"

// LLMOpts contains fields needed to connect to an LLM 
type LLMOpts struct {
	Query string
	Context any
	ApiKey string
	OrgId string
	ProjectId string
	Model string
	Temp string
}

// Connllm initializes connection to a LLM'S API parsing some specified LLMOpts
func  Connllm(opts LLMOpts) error {
	payload := map[string]interface{}{
		"model": opts.Model,
		"messages": []map[string]string{
			{"role": role, "content": opts.Query},
		},
		// "context": opts.context, //Mapper Schema
		// "temperature": opts.Temp,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("error marshaling json: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, llmurl, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	// Add headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", opts.ApiKey))
	req.Header.Set("OpenAI-Organization", opts.OrgId)
	req.Header.Set("OpenAI-Project", opts.ProjectId)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request: %v", err)
		
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("non-ok http status: %v", resp.Status)
		
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response body: %v", err)
		
	}

	fmt.Printf("Response: %s\n", body)
	return nil
}