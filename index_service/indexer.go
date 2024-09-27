package indexservice

import (
	"bytes"
	"encoding/gob"
	"search_engine/internal/kvdb"
	reverseindex "search_engine/internal/reverse_index"
	"search_engine/types"
	"search_engine/util"
	"strings"
	"sync/atomic"
)

// 外观Facade模式。把子系统封装到一起（这里是正排和倒排）
type Indexer struct {
	forwardindex kvdb.IKeyValueDB
	reverseIndex reverseindex.IReverseIndexer
	maxIntId     uint64
}

// 初始化索引：打开kv数据库，生成一个倒排索引跳表
// DataDir表示最后存数据的地方？
func (indexer *Indexer) Init(DocNumEstimate int, dbtype int, DataDir string) error {
	db, err := kvdb.GetKvDb(dbtype, DataDir) //调用工厂方法，打开本地的KV数据库
	if err != nil {
		return err
	}
	indexer.forwardindex = db
	indexer.reverseIndex = reverseindex.NewSKipListReverseIndex(DocNumEstimate)
	return nil
}

// 关闭索引
func (indexer *Indexer) Close() error {
	return indexer.forwardindex.Close() //db.Close()
}

// 系统重启后，直接从索引文件里加载数据
func (indexer *Indexer) LoadFromIndexFile() int {
	reader := bytes.NewReader([]byte{})
	n := indexer.forwardindex.IterDB(func(k, v []byte) error {
		reader.Reset(v) //reset成v
		decoder := gob.NewDecoder(reader)
		var doc types.Document
		err := decoder.Decode(&doc)
		if err != nil {
			util.Log.Printf("gob decode document failed：%s", err)
			return nil
		}
		indexer.reverseIndex.Add(doc)
		return err
	})
	util.Log.Printf("load %d data from forward index %s", n, indexer.forwardindex.GetDbPath())
	return int(n)
}

// 向索引中添加(亦是更新)文档(如果已存在，会先删除)
func (indexer *Indexer) AddDoc(doc types.Document) (int, error) {
	docId := strings.TrimSpace(doc.Id)
	if len(docId) == 0 {
		return 0, nil
	}
	//先从正排和倒排索引上将docId删除
	indexer.DeleteDoc(docId) //没有这个索引返回的是0

	doc.IntId = atomic.AddUint64(&indexer.maxIntId, 1) //写入索引时自动为文档生成IntId
	//写入正排索引
	var value bytes.Buffer
	encoder := gob.NewEncoder(&value)
	if err := encoder.Encode(doc); err == nil {
		indexer.forwardindex.Set([]byte(docId), value.Bytes())
	} else {
		return 0, err
	}

	//写入倒排索引
	indexer.reverseIndex.Add(doc)
	return 1, nil
}

// 从索引上删除文档
// 倒排和正排都要删
func (indexer *Indexer) DeleteDoc(docId string) int {
	n := 0
	forwardKey := []byte(docId)
	//先读正排索引，得到IntId和Keywords
	docBs, err := indexer.forwardindex.Get(forwardKey)
	if err == nil {
		reader := bytes.NewReader([]byte{})
		if len(docBs) > 0 {
			n = 1
			reader.Reset(docBs)
			decoder := gob.NewDecoder(reader)
			var doc types.Document
			err := decoder.Decode(&doc)
			if err == nil {
				// 遍历每一个keyword，从倒排索引上删除
				for _, kw := range doc.Keywords {
					indexer.reverseIndex.Delete(doc.IntId, kw)
				}
			}
		}
	}
	//从正排上删除
	indexer.forwardindex.Delete(forwardKey)
	return n
}

// 检索，返回文档列表
func (indexer *Indexer) Search(query *types.TermQuery, onFlag uint64, offFlag uint64, orFlags []uint64) []*types.Document {
	docIds := indexer.reverseIndex.Search(query, onFlag, offFlag, orFlags)
	if len(docIds) == 0 {
		return nil
	}
	keys := make([][]byte, 0, len(docIds))
	for _, docId := range docIds {
		keys = append(keys, []byte(docId))
	}
	docs, err := indexer.forwardindex.BatchGet(keys)
	if err != nil {
		util.Log.Printf("read kvdb failed: %s", err)
		return nil
	}
	result := make([]*types.Document, 0, len(docs))
	reader := bytes.NewReader([]byte{})
	for _, docBs := range docs {
		if len(docBs) > 0 {
			reader.Reset(docBs)
			decoder := gob.NewDecoder(reader)
			var doc types.Document
			err := decoder.Decode(&doc)
			if err == nil {
				result = append(result, &doc)
			}
		}
	}
	return result
}

// 索引里有几个document
func (indexer *Indexer) Count() int {
	n := 0
	indexer.forwardindex.IterKey(func(k []byte) error {
		n++
		return nil
	})
	return n
}
