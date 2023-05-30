package llm

import (
	"context"
)

// LLM 大语言模型
type LLM interface {
	// Embed 文本嵌入
	Embed(ctx context.Context, text string) ([]float32, error)
	// QA 回答问题
	QA(ctx context.Context, questionContext, question string) (string, error)
}
