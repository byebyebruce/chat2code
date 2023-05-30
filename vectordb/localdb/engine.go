package localdb

import (
	"container/heap"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/boltdb/bolt"
	"github.com/byebyebruce/chat2code/vectordb"
	"github.com/byebyebruce/chat2code/version"
	"github.com/coreos/go-semver/semver"
	"github.com/gogo/protobuf/proto"
	"github.com/sourcegraph/conc/pool"
)

const (
	metaBucketName  = "_meta"   // _meta->version
	chunkBucketName = "_chunks" // md5->doc
)

var _ vectordb.VectorDB = (*VectorDatabase)(nil)

// VectorDatabase 本地向量数据库
// 包含3个部分 meta、chunks、repo
// meta存储version等源信息
// chunks 存储md5->chunk的映射
// repo 存储的库包含的文档id和向量
type VectorDatabase struct {
	db *bolt.DB
}

func NewVectorSearch(path string) (*VectorDatabase, error) {
	// db 目录
	dbDir := filepath.Dir(path)
	if _, err := os.Stat(dbDir); err != nil {
		if os.IsNotExist(err) {
			if err := os.Mkdir(dbDir, 0755); err != nil {
				return nil, fmt.Errorf("create %s error:%v", path, err)
			}
		} else {
			return nil, err
		}
	}

	db, err := bolt.Open(path, os.ModePerm, &bolt.Options{Timeout: time.Second})
	if err != nil {
		return nil, err
	}
	err = db.Update(func(tx *bolt.Tx) error {
		bu, err := tx.CreateBucketIfNotExists([]byte(metaBucketName))
		if err != nil {
			return err
		}
		v := bu.Get([]byte(metaBucketName))
		if len(v) == 0 {
			return bu.Put([]byte(metaBucketName), []byte(version.Version))
		}
		oldVer := semver.New(string(v))
		newVer := semver.New(version.Version)
		if oldVer.Equal(*newVer) {
			return nil
		}
		if oldVer.Major != newVer.Major {
			return fmt.Errorf("version not match(%s,%s)delete file %s", oldVer.String(), newVer.String(), path)
		}
		if oldVer.Patch != newVer.Patch {
			// TODO upgrade
		}
		return bu.Put([]byte(metaBucketName), []byte(version.Version))
	})
	if err != nil {
		return nil, err
	}

	ret := &VectorDatabase{db: db}
	err = ret.CreateIfNotExists(chunkBucketName)
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (s *VectorDatabase) Close() error {
	return s.db.Close()
}

func (s *VectorDatabase) ListRepos(ctx context.Context) ([]string, error) {
	hide := map[string]struct{}{
		metaBucketName:  {},
		chunkBucketName: {},
	}
	var ret []string
	err := s.db.View(func(tx *bolt.Tx) error {
		tx.ForEach(func(name []byte, b *bolt.Bucket) error {
			repo := string(name)
			if _, ok := hide[repo]; ok {
				return nil
			}
			ret = append(ret, repo)
			return nil
		})
		return nil
	})
	return ret, err
}

func (s *VectorDatabase) CreateIfNotExists(repo string) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(repo))
		return err
	})
}

func (s *VectorDatabase) doc(md5 []byte) []byte {
	var b []byte
	s.db.View(func(tx *bolt.Tx) error {
		b = tx.Bucket([]byte(chunkBucketName)).Get(md5)
		return nil
	})
	return b
}

func (s *VectorDatabase) Upsert(ctx context.Context, repo string, v *vectordb.Upsert) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		bu := tx.Bucket([]byte(repo))
		b, err := proto.Marshal(v)
		if err != nil {
			return err
		}
		if err := bu.Put([]byte(v.ID), b); err != nil {
			return err
		}
		return tx.Bucket([]byte(chunkBucketName)).Put(v.MD5, v.Doc)
	})
}

func (s *VectorDatabase) Delete(ctx context.Context, repo string, ids ...string) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		bu := tx.Bucket([]byte(repo))
		for _, id := range ids {
			return bu.Delete([]byte(id))
		}
		return nil
	})
}

func (s *VectorDatabase) DeleteRepo(ctx context.Context, repo string) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		return tx.DeleteBucket([]byte(repo))
	})
	// TODO delete chunk
}

func (s *VectorDatabase) Range(ctx context.Context, repo string, f func(vector *vectordb.Vector)) error {
	return s.db.View(func(tx *bolt.Tx) error {
		return tx.Bucket([]byte(repo)).ForEach(func(k, v []byte) error {
			ret := &vectordb.Vector{}
			err := proto.Unmarshal(v, ret)
			if err != nil {
				return err
			}
			f(ret)
			return nil
		})
	})
}

func (s *VectorDatabase) Search(ctx context.Context, repo string, v []float32, topK int) ([]*vectordb.Match, error) {
	if topK <= 0 {
		return nil, fmt.Errorf("error topK %d", topK)
	}

	p := pool.New().WithMaxGoroutines(runtime.GOMAXPROCS(0))
	mu := sync.Mutex{}

	if topK == 1 {
		var m *vectordb.Match
		s.Range(ctx, repo, func(vector *vectordb.Vector) {
			p.Go(func() {
				score := Cosine(vector.Values, v)
				mu.Lock()
				if m == nil || score > m.Score {
					m = &vectordb.Match{Vector: *vector, Score: score}
				}
				mu.Unlock()
			})
		})
		p.Wait()
		m.Doc = s.doc(m.MD5)
		return []*vectordb.Match{m}, nil
	}

	var ret = &bigHeap{}
	s.Range(ctx, repo, func(vector *vectordb.Vector) {
		p.Go(func() {
			score := Cosine(vector.Values, v)
			mu.Lock()
			heap.Push(ret, &vectordb.Match{Vector: *vector, Score: score})
			if ret.Len() > topK {
				heap.Pop(ret)
			}
			mu.Unlock()
		})
	})
	p.Wait()
	for _, match := range *ret {
		match.Doc = s.doc(match.MD5)
	}
	return *ret, nil
}
