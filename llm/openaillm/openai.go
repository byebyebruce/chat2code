package openaillm

import (
	"context"
	"fmt"
	"strings"

	"github.com/byebyebruce/chat2code/llm"
	"github.com/sashabaranov/go-openai"
)

var _ llm.LLM = (*OpenAILLM)(nil)

// OpenAILLM 大语言模型，实现chat completion和embed
type OpenAILLM struct {
	cli   *openai.Client
	model string
}

// NewLLM 构造
func NewLLM(apiKey string, apiBase string, model string) *OpenAILLM {
	if model == "" {
		model = openai.GPT3Dot5Turbo
	}
	var cfg openai.ClientConfig
	if strings.HasPrefix(apiKey, "ak-") { // open ai key
		cfg = openai.DefaultConfig(apiKey)
		if len(apiBase) > 0 {
			cfg.BaseURL = apiBase
		}
	} else { // azure key
		cfg = openai.DefaultAzureConfig(apiKey, apiBase)
	}
	cli := openai.NewClientWithConfig(cfg)
	return &OpenAILLM{cli: cli, model: model}
}

// Embed 嵌入
func (l *OpenAILLM) Embed(ctx context.Context, text string) ([]float32, error) {
	resp, err := l.cli.CreateEmbeddings(ctx, openai.EmbeddingRequest{
		Input: []string{text},
		Model: openai.AdaEmbeddingV2,
	})
	if err != nil {
		return nil, err
	}
	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("no result")
	}
	return resp.Data[0].Embedding, nil
}

// QA 问答
func (l *OpenAILLM) QA(ctx context.Context, questionContext, question string) (string, error) {
	req := openai.ChatCompletionRequest{
		Model: l.model,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleUser,
				Content: fmt.Sprintf(prompt_template, questionContext, question),
			},
		},
	}

	resp, err := l.cli.CreateChatCompletion(ctx, req)
	if err != nil {
		return "", err
	}
	return resp.Choices[0].Message.Content, nil
}
