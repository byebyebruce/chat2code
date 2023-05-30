//go:generate protoc --proto_path=./ --gogo_out=paths=source_relative:./ types.proto
package vectordb

import (
	"context"
)

// Match 匹配结果
type Match struct {
	Vector
	Score float32
	Doc   []byte
}

// Upsert 插入/更新
type Upsert struct {
	Vector
	Doc []byte
}

// VectorDB 向量数据库
type VectorDB interface {
	// ListRepos list repos
	ListRepos(ctx context.Context) ([]string, error)

	// DeleteRepo delete repo
	DeleteRepo(ctx context.Context, repos string) error

	// Upsert 插入or更新
	Upsert(ctx context.Context, repo string, v *Upsert) error

	// Delete 删除
	Delete(ctx context.Context, repo string, id ...string) error

	// Range 遍历所有
	Range(ctx context.Context, repo string, f func(*Vector)) error

	// Search 搜索向量相似度最高的topk
	Search(ctx context.Context, repo string, v []float32, topK int) ([]*Match, error)
}
