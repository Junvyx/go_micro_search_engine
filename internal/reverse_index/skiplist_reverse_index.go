package reverseindex

import (
	"runtime"
	"search_engine/types"
	"search_engine/util"
	"sync"

	"github.com/huandu/skiplist"
	farmhash "github.com/leemcloughlin/gofarmhash"
)

// 倒排索引整体上是一个map，map的value是一个List
// concurrenthashmap 由keywords和对应的小map（value）组成；
// 每个map里的key是IntId，值是Id和BitsFeature（下面顶一个SkipListValue）
type SkipListReverseIndex struct {
	table *util.CocurrentHashMap //分段map，并发安全。map里面是跳表
	locks []sync.RWMutex
}

func NewSKipListReverseIndex(DocNumEstimate int) *SkipListReverseIndex {
	indexer := new(SkipListReverseIndex)
	//根据CPU数量来确定小mps的数量; seg,cap
	indexer.table = util.NewConcurrentHashMap(runtime.NumCPU(), DocNumEstimate)
	indexer.locks = make([]sync.RWMutex, 1000)
	return indexer
}

func (indexer SkipListReverseIndex) getLock(key string) *sync.RWMutex {
	n := int(farmhash.Hash32WithSeed([]byte(key), 0))
	return &indexer.locks[n%len(indexer.locks)]
}

type SkipListValue struct {
	Id          string
	BitsFeature uint64
}

// 添加一个doc
func (indexer *SkipListReverseIndex) Add(doc types.Document) {
	for _, keyword := range doc.Keywords {
		//先找到key上锁
		key := keyword.ToString()
		lock := indexer.getLock(key)
		lock.Lock()
		sklValue := SkipListValue{doc.Id, doc.BitsFeature}
		if value, exists := indexer.table.Get(key); exists {
			value.(*skiplist.SkipList).Set(doc.IntId, sklValue) //IntId作为SkipList的key，而value里则包含了业务侧的文档id和BitsFeature
		} else {
			list := skiplist.New(skiplist.Uint64)
			list.Set(doc.IntId, sklValue)
			indexer.table.Set(key, list)
		}
		// util.Log.Printf("add key %s value %d to reverse index\n", key, DocId)
		lock.Unlock()
	}
}

// 从key上删除对应的doc
func (indexer *SkipListReverseIndex) Delete(IntId uint64, keyword *types.Keyword) {
	key := keyword.ToString()
	lock := indexer.getLock(key)
	lock.Lock()
	if value, exists := indexer.table.Get(key); exists {
		value.(*skiplist.SkipList).Remove(IntId)
	}
	lock.Unlock()
}

// 求多个Skiplist的交集
// key是intid。值是id+bits
func IntersectionOfSkipList(lists ...*skiplist.SkipList) *skiplist.SkipList {
	if len(lists) == 0 {
		return nil
	}

	if len(lists) == 1 {
		return lists[0]
	}

	result := skiplist.New(skiplist.Uint64)
	//curr用来保存每个lists当前要对比的节点
	//curr是用来遍历每个list的节点的
	currNodes := make([]*skiplist.Element, len(lists))
	for i, list := range lists {
		if list.Len() == 0 || list == nil {
			return nil
		}
		currNodes[i] = list.Front() //初始是每个跳表的头结点
	}

	for {
		maxList := make(map[int]struct{}, len(currNodes)) //存储每个跳表当前节点的最大值，最大值有可能有多个
		var maxValue uint64 = 0                           //下标是从0开始的
		for i, node := range currNodes {
			if node.Key().(uint64) > maxValue {
				maxValue = node.Key().(uint64)
				maxList = map[int]struct{}{i: {}} //有新的最大值则用新的map覆盖前面的，最终每个最大值的节点都会有一个key:value（索引：空结构体）
			} else if node.Key().(uint64) == maxValue {
				maxList[i] = struct{}{}
			}
		}
		//如果所有node都一样大，那maxList的长度就和currNodes一样大，有那么多的空结构体
		//则诞生一个交集
		if len(maxList) == len(currNodes) {
			result.Set(currNodes[0].Key(), currNodes[0].Value)
			//所有节点向后移一位
			for i, node := range currNodes {
				currNodes[i] = node.Next() //不能用node=node.Next()，因为for range取得的是值拷贝
				if currNodes[i] == nil {
					return result
				}
			}
		} else {
			for i, node := range currNodes {
				if _, exists := maxList[i]; !exists {
					currNodes[i] = node.Next()
					if currNodes[i] == nil {
						return result
					}
				}
			}
		}
	}
}

// 求多个SkipList的并集
func UnionsetOfSkipList(lists ...*skiplist.SkipList) *skiplist.SkipList {
	if len(lists) == 0 {
		return nil
	}

	if len(lists) == 1 {
		return lists[0]
	}

	result := skiplist.New(skiplist.Uint64)
	keySet := make(map[any]struct{}, 1000)
	//遍历每个list就行
	for _, list := range lists {
		if list == nil {
			continue
		}
		node := list.Front()

		for node != nil {
			if _, exists := keySet[node.Key()]; !exists {
				result.Set(node.Key(), node.Value)
				keySet[node.Key()] = struct{}{}
			}
			node = node.Next()
		}
	}
	return result
}

// 按照bits特征进行过滤
func (indexer SkipListReverseIndex) FilterByBits(bits uint64, onFlag uint64, offFlag uint64, orFlag []uint64) bool {
	//onFlag所有bit必须全部命中
	if bits&onFlag != onFlag {
		return false
	}

	//offFlag所有bit必须全部不命中
	if bits&offFlag != 0 {
		return false
	}

	//多个orFlags必须全部命中??
	for _, orFlag := range orFlag {
		if orFlag > 0 && bits&orFlag <= 0 { //单个orFlag只有一个bit命中即可
			return false
		}
	}
	return true
}

// 搜索，返回skiplist
func (indexer SkipListReverseIndex) search(q *types.TermQuery, onFlag uint64, offFlag uint64, orFlags []uint64) *skiplist.SkipList {
	if q.Keyword != nil {
		Keyword := q.Keyword.ToString()
		if value, exists := indexer.table.Get(Keyword); exists {
			result := skiplist.New(skiplist.Uint64)
			list := value.(*skiplist.SkipList)
			//util.Log.Printf("retrive %d docs by key %s", list.Len(), Keyword)
			node := list.Front()
			for node != nil {
				intId := node.Key().(uint64)
				skv, _ := node.Value.(SkipListValue)
				flag := skv.BitsFeature
				if intId > 0 && indexer.FilterByBits(flag, onFlag, offFlag, orFlags) {
					result.Set(intId, skv)
				}
				node = node.Next()
			}
			return result
		}
	} else if len(q.Must) > 0 {
		results := make([]*skiplist.SkipList, 0, len(q.Must))
		for _, q := range q.Must {
			results = append(results, indexer.search(q, onFlag, offFlag, orFlags))
		}
		return IntersectionOfSkipList(results...)
	} else if len(q.Should) > 0 {
		results := make([]*skiplist.SkipList, 0, len(q.Should))
		for _, q := range q.Should {
			results = append(results, indexer.search(q, onFlag, offFlag, orFlags))
		}
		return UnionsetOfSkipList(results...)
	}
	return nil
}

// 搜索，返回docId
func (indexer SkipListReverseIndex) Search(query *types.TermQuery, onFlag uint64, offFlag uint64, orFlags []uint64) []string {
	result := indexer.search(query, onFlag, offFlag, orFlags)
	if result == nil {
		return nil
	}
	arr := make([]string, 0, result.Len()) //跳表的长度
	node := result.Front()
	for node != nil {
		skv, _ := node.Value.(SkipListValue)
		arr = append(arr, skv.Id)
		node = node.Next()
	}
	return arr
}
