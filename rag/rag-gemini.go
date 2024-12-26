package rag

import (
	"context"
	"fmt"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

type GeminiLLM struct {
	Opts     LLMOpts
	Query    string
	Response string
}

// NewGeminiLLM is used to initalize a LLM that communicates with Google's Gemini API.
func NewGeminiLLM(opts LLMOpts) LLM {
	return &GeminiLLM{
		Opts: opts,
	}
}

func (llm *GeminiLLM) GenerateQuery(que string) (string, error) {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(llm.Opts.ApiKey))
	if err != nil {
		return llm.Query, nil
	}
	defer client.Close()

	model := client.GenerativeModel(llm.Opts.Model)
	cs := model.StartChat()

	cs.History = []*genai.Content{
		{
			Parts: []genai.Part{
				genai.Text("Generate a sql query from the next stream of input or text"),
			},
			Role: "user",
		},
		{
			Parts: []genai.Part{
				genai.Text("Only SELECT queries or queries to read data should be generated"),
			},
			Role: "user",
		},
		{
			Parts: []genai.Part{
				genai.Text(fmt.Sprintf("The Schema for the database is in the form: %v", llm.Opts.Context)),
			},
			Role: "user",
		},
		{
			Parts: []genai.Part{
				genai.Text("Only queries based on the database schema should be generated"),
			},
			Role: "user",
		},
		{
			Parts: []genai.Part{
				genai.Text("Omit fields or columns with sensitive data such as password, hashed_password or similar fields no matter the condtions stated in corresponding statements."),
			},
			Role: "user",
		},
		{
			Parts: []genai.Part{
				genai.Text("If none of the conditions are satisfied, return a custom error response"),
			},
			Role: "user",
		},
		{
			Parts: []genai.Part{
				genai.Text("If all the conditions are satisfied, return only the SQL query as a response"),
			},
			Role: "user",
		},
		{
			Parts: []genai.Part{
				genai.Text("Return the respone as a plain text rather than a block of code and remove the indentations~"),
			},
			Role: "user",
		},
	}

	res, err := cs.SendMessage(ctx, genai.Text(que))
	if err != nil {
		return llm.Query, nil
	}
	llm.Query, err = getGeminiResponse(res)
	if err != nil {
		return llm.Query, nil
	}

	return llm.Query, nil
}

func (llm *GeminiLLM) GenerateResponse(data interface{}, que string) (string, error) {
	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(llm.Opts.ApiKey))
	if err != nil {
		return llm.Response, nil
	}
	defer client.Close()

	model := client.GenerativeModel(llm.Opts.Model)
	cs := model.StartChat()

	cs.History = []*genai.Content{
		{
			Parts: []genai.Part{
				genai.Text("Generate a summary from the next stream of input or text"),
			},
			Role: "user",
		},
		{
			Parts: []genai.Part{
				genai.Text(fmt.Sprintf("Use these data: %v retrieved from the database in a conversational manner", data)),
			},
			Role: "user",
		},
		{
			Parts: []genai.Part{
				genai.Text(fmt.Sprintf("Use this as context for the data returned: %v", que)),
			},
			Role: "user",
		},
	}

	res, err := cs.SendMessage(ctx, genai.Text("What is the summary of the data?"))
	if err != nil {
		return llm.Response, nil
	}
	llm.Response, err = getGeminiResponse(res)
	if err != nil {
		return llm.Response, nil
	}

	return llm.Response, nil
}

func getGeminiResponse(resp *genai.GenerateContentResponse) (string, error) {
	res := ""
	for _, cand := range resp.Candidates {
		if cand.Content != nil {
			for _, part := range cand.Content.Parts {
				res = fmt.Sprintf("%v", part)
			}
		}
	}
	return res, nil
}
