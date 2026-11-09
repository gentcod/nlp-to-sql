package rag

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitLLM_SupportsMuse(t *testing.T) {
	llm := InitLLM("muse", LLMOpts{
		ApiKey: "test-muse-key",
		Model:  "muse-wonder-1.0",
	})
	require.NotNil(t, llm)
	_, ok := llm.(*MuseLLM)
	assert.True(t, ok)

	// Case-insensitive and whitespace-tolerant
	llmUpper := InitLLM("  MUSE ", LLMOpts{})
	require.NotNil(t, llmUpper)
	_, ok = llmUpper.(*MuseLLM)
	assert.True(t, ok)
}

func TestMuseLLM_ResponsesAPIGenerateQueryAndResponse(t *testing.T) {
	var capturedAuth string
	var capturedModel string
	var capturedStore bool
	var capturedSchemaName string

	var capturedInput string

	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		assert.Equal(t, "/v1/responses", r.URL.Path)

		var req map[string]any
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)

		capturedModel, _ = req["model"].(string)
		capturedStore, _ = req["store"].(bool)
		inputStr, _ := req["input"].(string)
		require.NotEmpty(t, inputStr)
		capturedInput = inputStr

		textCfg, ok := req["text"].(map[string]any)
		require.True(t, ok)
		formatCfg, ok := textCfg["format"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "json_schema", formatCfg["type"])
		assert.Equal(t, true, formatCfg["strict"])
		capturedSchemaName, _ = formatCfg["name"].(string)

		// Verify enforceStrict added additionalProperties: false and required array
		schemaDoc, ok := formatCfg["schema"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, false, schemaDoc["additionalProperties"])
		assert.NotEmpty(t, schemaDoc["required"])

		var outputText string
		if strings.Contains(capturedSchemaName, "SQL") {
			outputText = `{"sql":"SELECT id, username FROM users;"}`
		} else {
			outputText = `{"summary":"There is 1 user named alice."}`
		}

		w.Header().Set("Content-Type", "application/json")
		resp := map[string]any{
			"status": "completed",
			"output": []map[string]any{
				{
					"type": "message",
					"content": []map[string]any{
						{
							"type": "output_text",
							"text": outputText,
						},
					},
				},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer mockServer.Close()

	llm := InitLLM(LLMTypeMuse, LLMOpts{
		ApiKey:  "muse-secret-token",
		BaseUrl: mockServer.URL + "/v1/responses",
		Model:   "muse-wonder-1.0",
		Temp:    "0.2",
		Context: map[string]map[string]string{
			"users": {
				"id":               "uuid",
				"username":         "varchar",
				"harshed_password": "varchar",
				"api_token":        "varchar",
			},
		},
	})
	require.NotNil(t, llm)

	sqlQuery, err := llm.GenerateQuery("List all user ids and usernames")
	require.NoError(t, err)
	assert.Equal(t, "SELECT id, username FROM users;", sqlQuery)
	assert.Equal(t, "Bearer muse-secret-token", capturedAuth)
	assert.Equal(t, "muse-wonder-1.0", capturedModel)
	assert.False(t, capturedStore)
	// Verify that sensitive columns were stripped from the schema context before reaching the LLM prompt
	assert.NotContains(t, capturedInput, "harshed_password:")
	assert.NotContains(t, capturedInput, "api_token:")

	summary, err := llm.GenerateResponse([]map[string]interface{}{{"id": "1", "username": "alice"}}, "List all user ids and usernames")
	require.NoError(t, err)
	assert.Equal(t, "There is 1 user named alice.", summary)
}
