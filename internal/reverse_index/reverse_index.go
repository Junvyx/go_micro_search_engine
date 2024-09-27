package reverseindex

import "search_engine/types"

//倒排索引的接口，实现三个方法
type IReverseIndexer interface {
	Add(doc types.Document)                                                              //添加文件
	Delete(IntId uint64, keyword *types.Keyword)                                         //根据IntId删除关键词及对应的文件
	Search(q *types.TermQuery, onFlag uint64, offFlag uint64, orFlags []uint64) []string //查找,返回业务侧文档ID
}
