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

	cc, err := client.Caches.Create(ctx, "gemini-1.5-flash-001", &genai.CreateCachedContentConfig{
		SystemInstruction: genai.NewContentFromText("You are an expert analyzing transcripts.", genai.RoleUser),
		Contents:          []*genai.Content{genai.NewContentFromText(queryInstructions(Opts.Context), genai.RoleUser)},
	})
	if err != nil {
		return nil, err
	}

	return &Model{client: client, model: "gemini-1.5-flash-001", cachedName: cc.Name}, nil
}

func (llm *GeminiLLM) GenerateQuery(que string) (string, error) {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{APIKey: llm.Opts.ApiKey})
	if err != nil {
		return llm.Query, err
	}
	res, err := client.Models.GenerateContent(ctx, llm.Opts.Model,
		[]*genai.Content{genai.NewContentFromText(que, genai.RoleUser)},
		&genai.GenerateContentConfig{
			SystemInstruction: genai.NewContentFromText(queryInstructions(llm.Opts.Context), genai.RoleUser),
		})
	if err != nil {
		return llm.Query, err
	}
	llm.Query, err = getGeminiResponse(res)
	if err != nil {
		return llm.Query, err
	}

	return llm.Query, nil
}

func (llm *GeminiLLM) GenerateResponse(data interface{}, que string) (string, error) {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{APIKey: llm.Opts.ApiKey})
	if err != nil {
		return llm.Response, err
	}
	res, err := client.Models.GenerateContent(ctx, llm.Opts.Model,
		[]*genai.Content{genai.NewContentFromText("What is the summary of the data?", genai.RoleUser)},
		&genai.GenerateContentConfig{
			SystemInstruction: genai.NewContentFromText(strings.Join([]string{
				"Generate a summary from the next stream of input or text",
				fmt.Sprintf("Use these data: %v retrieved from the database in a conversational manner", data),
				fmt.Sprintf("Use this as context for the data returned: %v", que),
			}, "\n"), genai.RoleUser),
		})
	if err != nil {
		return llm.Response, err
	}
	llm.Response, err = getGeminiResponse(res)
	if err != nil {
		return llm.Response, err
	}

	return llm.Response, nil
}

func getGeminiResponse(resp *genai.GenerateContentResponse) (string, error) {
	return resp.Text(), nil
}

func queryInstructions(schema any) string {
	return strings.Join([]string{
		"Generate a sql query from the next stream of input or text",
		"Only SELECT queries or queries to read data should be generated",
		fmt.Sprintf("The Schema for the database is in the form: %v", schema),
		"Only queries based on the database schema should be generated",
		"Omit fields or columns with sensitive data such as password, hashed_password or similar fields no matter the conditions stated in corresponding statements.",
		"If none of the conditions are satisfied, return a custom error response",
		"If all the conditions are satisfied, return only the SQL query as a response",
		"Return the response as plain text rather than a block of code and remove the indentations",
	}, "\n")
}

func (llm *GeminiLLM) GetModelContext() (interface{}, error) {
	model, err := initGenModelWithCachedContent(llm.Opts)
	if err != nil {
		return nil, err
	}

	return model, err
}

func (model *Model) GenerateCachedResponse(prompt string) (string, error) {
	ctx := context.Background()
	res, err := model.client.Models.GenerateContent(ctx, model.model,
		[]*genai.Content{genai.NewContentFromText(prompt, genai.RoleUser)},
		&genai.GenerateContentConfig{CachedContent: model.cachedName})
	if err != nil {
		return "", err
	}
	_, deleteErr := model.client.Caches.Delete(ctx, model.cachedName, nil)
	if deleteErr != nil {
		return "", deleteErr
	}

	resp, err := getGeminiResponse(res)
	if err != nil {
		return "", err
	}
	return resp, nil
}
