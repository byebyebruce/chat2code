package chat2code

import (
	"context"
	"fmt"
	"time"

	"github.com/byebyebruce/chat2code/llm"
	"github.com/byebyebruce/chat2code/pkg/util"
	"github.com/byebyebruce/chat2code/vectordb"
	"github.com/fatih/color"
	"github.com/sourcegraph/conc/pool"
)

const (
	metaFile  = `file`  // meta key
	metaBlock = `block` // meta key
)

// Chunk 文本块
type Chunk struct {
	ID    string
	File  string
	Index int
	Text  string
	MD5   []byte
}

// Answer 回答
type Answer struct {
	Answer  string
	File    string
	Context string
}

// Chat2Code 和代码对话模块
type Chat2Code struct {
	llm   llm.LLM           // llm
	vecDB vectordb.VectorDB // 向量db
}

// NewChat2Code 构造
func NewChat2Code(db vectordb.VectorDB, llm llm.LLM) (*Chat2Code, error) {
	return &Chat2Code{
		vecDB: db,
		llm:   llm,
	}, nil
}

// Load 加载文本块
func (c *Chat2Code) Load(ctx context.Context, repo string, chunks map[string]*Chunk, embedThread int) error {
	var cleanup []string
	c.vecDB.Range(ctx, repo, func(vector *vectordb.Vector) {
		doc, ok := chunks[vector.ID]
		if !ok {
			cleanup = append(cleanup, vector.ID)
			return
		}
		if util.Md5Equal(doc.MD5, vector.MD5) {
			delete(chunks, doc.ID)
			return
		}
	})

	// clean up
	if len(cleanup) > 0 {
		fmt.Println("clean up... ", len(cleanup))
		if err := c.vecDB.Delete(ctx, repo, cleanup...); err != nil {
			return err
		}
	}

	p := pool.New().WithMaxGoroutines(embedThread).WithErrors() // TODO,接口掉太快会返回错误
	for _, s := range chunks {
		fmt.Println("embedding", color.BlueString(s.File), s.Index)
		doc := s
		p.Go(func() error {
			embedCtx, cancel := context.WithTimeout(ctx, time.Second*10)
			defer cancel()
			v, err := c.llm.Embed(embedCtx, doc.Text)
			if err != nil {
				fmt.Println("Embedding", doc.File, err)
				return err
			}
			err = c.vecDB.Upsert(ctx, repo, &vectordb.Upsert{
				Vector: vectordb.Vector{
					ID:     doc.ID,
					Values: v,
					MD5:    doc.MD5,
					Meta:   map[string]string{metaFile: doc.File, metaBlock: fmt.Sprint(doc.Index)},
				},
				Doc: []byte(doc.Text),
			})
			if err != nil {
				return err
			}
			return nil
		})
	}

	return p.Wait()
}

// Answer 回答问题
func (c *Chat2Code) Answer(ctx context.Context, repo string, question string, threshold /*[0~1]*/ float32) (*Answer, error) {
	vec, err := c.llm.Embed(ctx, question)
	if err != nil {
		return nil, err
	}
	vs, err := c.vecDB.Search(ctx, repo, vec, 1)
	if err != nil {
		return nil, err
	}
	if len(vs) == 0 {
		return nil, nil
	}

	v := vs[0]
	if v.Score < threshold {
		return nil, nil
	}
	doc := v.Doc
	if len(doc) == 0 {
		return nil, fmt.Errorf("no doc %s", v.ID)
	}
	docStr := string(doc)
	answer, err := c.llm.QA(ctx, docStr, question)
	if err != nil {
		return nil, err
	}
	return &Answer{
		Answer:  answer,
		Context: docStr,
		File:    v.Meta[metaFile],
	}, nil

}
