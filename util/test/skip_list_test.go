package test

import (
	"fmt"
	"testing"

	"github.com/huandu/skiplist"
)

func TestSkipList(t *testing.T) {
	//新建跳表
	list := skiplist.New(skiplist.Int32)
	//向跳表添加元素
	list.Set(24, 31) //skiplist是一个按key排序好的map
	list.Set(24, 40) //键覆盖
	list.Set(12, 40)
	list.Set(18, 3)
	list.Remove(12)
	if value, ok := list.GetValue(18); ok {
		fmt.Println(value)
	}

	//自动按key排好序
	fmt.Println("-------------------")
	node := list.Front()
	for node != nil {
		fmt.Println(node.Key(), node.Value)
		node = node.Next()
	}
}

// go test -v ./util/test -run=^TestSkipList$ -count=1
