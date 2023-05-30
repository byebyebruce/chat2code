package text_splitter

import (
	"fmt"
	"io/ioutil"
	"runtime"
	"sync"

	"github.com/byebyebruce/chat2code"
	"github.com/byebyebruce/chat2code/pkg/util"
	"github.com/sourcegraph/conc/pool"
	"github.com/tmc/langchaingo/exp/text_splitters"
)

// SplitText 分隔
func SplitText(fileName, text string, chunk_size int, overlap int) ([]*chat2code.Chunk, error) {
	cs := text_splitters.NewRecursiveCharactersSplitter()
	cs.ChunkOverlap = overlap
	cs.ChunkSize = chunk_size

	chunks, err := cs.SplitText(text)
	if err != nil {
		return nil, err
	}
	var allChunks = make([]*chat2code.Chunk, 0, len(chunks))
	for i, chunk := range chunks {
		md5 := util.Md5Hash(chunk)
		id := fmt.Sprintf("%s_%d", fileName, i)
		allChunks = append(allChunks, &chat2code.Chunk{
			ID:    id,
			File:  fileName,
			MD5:   md5,
			Text:  chunk,
			Index: i,
		})
	}
	return allChunks, nil
}

// SplitFiles 分隔
func SplitFiles(fs []string, chunk_size, overlap int) (map[string]*chat2code.Chunk, error) {
	allChunks := make(map[string]*chat2code.Chunk)
	cs := text_splitters.NewRecursiveCharactersSplitter()
	cs.ChunkOverlap = overlap
	cs.ChunkSize = chunk_size
	mu := sync.Mutex{}
	p := pool.New().WithMaxGoroutines(runtime.GOMAXPROCS(0)).WithErrors()

	for _, f := range fs {
		fname := f
		p.Go(func() error {
			b, err := ioutil.ReadFile(fname)
			if err != nil {
				return err
			}
			chunks, err := SplitText(fname, string(b), chunk_size, overlap)
			if err != nil {
				return err
			}
			for _, chunk := range chunks {
				mu.Lock()
				allChunks[chunk.ID] = chunk
				mu.Unlock()
			}
			return nil
		})
	}

	return allChunks, p.Wait()
}
