package util

import (
	"sync"

	farmhash "github.com/leemcloughlin/gofarmhash"
	"golang.org/x/exp/maps"
)

// 自行实现支持并发读写的map。key是string，value是any
type CocurrentHashMap struct {
	mps   []map[string]any //一个大map分为若干个小map，增加读写效率
	seg   int              //小map的个数
	locks []sync.RWMutex   //一个小map对应一个锁
	seed  uint32           //每次执行farmhash统一的seed
}

// 构造函数
// cap预估总map中容纳多少元素，seg内部包含几个小map
func NewConcurrentHashMap(seg, cap int) *CocurrentHashMap {
	mps := make([]map[string]any, seg)
	locks := make([]sync.RWMutex, seg)

	for i := 0; i < seg; i++ {
		mps[i] = make(map[string]any, cap/seg) //每个mp里包含cap/seg个键值对
	}

	return &CocurrentHashMap{
		mps:   mps,
		seg:   seg,
		seed:  0,
		locks: locks,
	}
}

// 判断key对应到哪个小map
func (m *CocurrentHashMap) getSegIndex(key string) int {
	hash := int(farmhash.Hash32WithSeed([]byte(key), m.seed)) //将字符串key转换为byte切片，哈希后再转为int
	return hash % m.seg                                       //对哈希值用map数取模
}

// 写入<key, value>
func (m *CocurrentHashMap) Set(key string, value any) {
	index := m.getSegIndex(key)
	m.locks[index].Lock()
	defer m.locks[index].Unlock()
	m.mps[index][key] = value
}

// 根据key读取value
func (m *CocurrentHashMap) Get(key string) (any, bool) {
	index := m.getSegIndex(key)
	m.locks[index].Lock()
	defer m.locks[index].Unlock()
	value, exists := m.mps[index][key]
	return value, exists
}

// 迭代器模式
// 需要有统一的方法（一般是Next)来遍历容器
type ConcurrentHashMapIterator struct {
	cm       *CocurrentHashMap
	keys     [][]string //map没有顺序，可以把keys保存在数组
	rowIndex int
	colIndex int
}

func (m *CocurrentHashMap) CreateIterator() *ConcurrentHashMapIterator {
	keys := make([][]string, 0, len(m.mps))
	for _, mp := range m.mps {
		row := maps.Keys(mp) //maps.Keys方法提取每个key,返回[]string
		keys = append(keys, row)
	}

	return &ConcurrentHashMapIterator{
		cm:       m,
		keys:     keys,
		rowIndex: 0,
		colIndex: 0,
	}
}

type MapEntry struct {
	Key   string
	Value any
}

func (iter *ConcurrentHashMapIterator) Next() *MapEntry {

	row := iter.keys[iter.rowIndex]
	if len(row) == 0 {
		iter.rowIndex += 1
		return iter.Next()
	}

	key := row[iter.colIndex]
	value, _ := iter.cm.Get(key)
	if iter.colIndex >= len(row)-1 {
		iter.colIndex = 0
		iter.rowIndex += 1
	} else {
		iter.colIndex += 1
	}
	return &MapEntry{key, value}
}

// 定义一个接口，所有有Next的对象都进来
// 因为也许不止cm需要迭代器模式
type MapIterator interface {
	Next() *MapEntry
}
