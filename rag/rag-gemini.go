package rag

import (
	"context"
	"fmt"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

type GeminiLLM struct {
	Query string
	Opts  LLMOpts
}

func NewGeminiLLM(opts LLMOpts) LLM {
	return &GeminiLLM{
		Opts: opts,
	}
}

func (llm *GeminiLLM) GenerateQuery() (string, error) {
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
				genai.Text(fmt.Sprintf("Generate a sql query for a %s database from the next stream of input or text", llm.Opts.DBType)),
			},
			Role: "user",
		},
		{
			Parts: []genai.Part{
				genai.Text("Only SELECT queries or queries to read data are allowed"),
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

	res, err := cs.SendMessage(ctx, genai.Text(llm.Opts.Query))
	if err != nil {
		return llm.Query, nil
	}
	llm.Query, err = getGeminiResponse(res)
	if err != nil {
		return llm.Query, nil
	}

	fmt.Println(llm.Query)

	return llm.Query, nil
}

func (llm *GeminiLLM) GenerateResponse(data any) {

}

func getGeminiResponse(resp *genai.GenerateContentResponse) (string, error) {
	res := ""
	for _, cand := range resp.Candidates {
		if cand.Content != nil {
			for _, part := range cand.Content.Parts {
				// fmt.Println("res", part)
				res = fmt.Sprintf("%v", part)
				fmt.Println(validQuery(res))

			}
		}
	}
	return res, nil
}
