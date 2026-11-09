package rag

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/genai"
)

type GeminiLLM struct {
	Opts     LLMOpts
	Query    string
	Response string
	Model    string
}

type Model struct {
	client     *genai.Client
	model      string
	cachedName string
}

// NewGeminiLLM is used to initalize a LLM that communicates with Google's Gemini API.
func NewGeminiLLM(opts LLMOpts) LLM {
	return &GeminiLLM{
		Opts: opts,
	}
}

func initGenModelWithCachedContent(Opts LLMOpts) (*Model, error) {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{APIKey: Opts.ApiKey})
	if err != nil {
		return nil, err
	}

	modelName := Opts.Model
	if modelName == "" {
		modelName = "gemini-1.5-flash-001"
	}

	cc, err := client.Caches.Create(ctx, modelName, &genai.CreateCachedContentConfig{
		SystemInstruction: genai.NewContentFromText("You are an expert analyzing database schemas and SQL queries.", genai.RoleUser),
		Contents:          []*genai.Content{genai.NewContentFromText(queryInstructions(Opts.Context), genai.RoleUser)},
	})
	if err != nil {
		return nil, err
	}

	return &Model{client: client, model: modelName, cachedName: cc.Name}, nil
}

func (llm *GeminiLLM) GenerateQuery(que string) (string, error) {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{APIKey: llm.Opts.ApiKey})
	if err != nil {
		return "", err
	}
	modelName := llm.Opts.Model
	if modelName == "" {
		modelName = "gemini-1.5-flash-001"
	}
	res, err := client.Models.GenerateContent(ctx, modelName,
		[]*genai.Content{genai.NewContentFromText(que, genai.RoleUser)},
		&genai.GenerateContentConfig{
			SystemInstruction: genai.NewContentFromText(queryInstructions(llm.Opts.Context), genai.RoleUser),
		})
	if err != nil {
		return "", err
	}
	query, err := getGeminiResponse(res)
	if err != nil {
		return "", err
	}

	llm.Query = query
	return query, nil
}

func (llm *GeminiLLM) GenerateResponse(data interface{}, que string) (string, error) {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{APIKey: llm.Opts.ApiKey})
	if err != nil {
		return "", err
	}
	modelName := llm.Opts.Model
	if modelName == "" {
		modelName = "gemini-1.5-flash-001"
	}
	res, err := client.Models.GenerateContent(ctx, modelName,
		[]*genai.Content{genai.NewContentFromText("What is the summary of the data?", genai.RoleUser)},
		&genai.GenerateContentConfig{
			SystemInstruction: genai.NewContentFromText(strings.Join([]string{
				"Generate a summary from the next stream of input or text",
				fmt.Sprintf("Use these data: %v retrieved from the database in a conversational manner", data),
				fmt.Sprintf("Use this as context for the data returned: %v", que),
			}, "\n"), genai.RoleUser),
		})
	if err != nil {
		return "", err
	}
	resp, err := getGeminiResponse(res)
	if err != nil {
		return "", err
	}

	llm.Response = resp
	return resp, nil
}

func getGeminiResponse(resp *genai.GenerateContentResponse) (string, error) {
	return resp.Text(), nil
}

func (llm *GeminiLLM) GetModelContext() (interface{}, error) {
	model, err := initGenModelWithCachedContent(llm.Opts)
	if err != nil {
		return nil, err
	}

	return model, err
}

// GenerateCachedResponse generates content using the remote Gemini cache and guarantees cache cleanup on exit.
func (model *Model) GenerateCachedResponse(prompt string) (string, error) {
	ctx := context.Background()
	defer func() {
		if model.client != nil && model.cachedName != "" {
			_, _ = model.client.Caches.Delete(ctx, model.cachedName, nil)
		}
	}()

	res, err := model.client.Models.GenerateContent(ctx, model.model,
		[]*genai.Content{genai.NewContentFromText(prompt, genai.RoleUser)},
		&genai.GenerateContentConfig{CachedContent: model.cachedName})
	if err != nil {
		return "", err
	}

	resp, err := getGeminiResponse(res)
	if err != nil {
		return "", err
	}
	return resp, nil
}
