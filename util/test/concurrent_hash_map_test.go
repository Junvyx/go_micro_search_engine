package test

import (
	"math/rand"
	"search_engine/util"
	"strconv"
	"sync"
	"testing"
)

//自己实现的支持并发的map和标准库sync.map对比

var conMp = util.NewConcurrentHashMap(8, 1000) //8个小map，容量共1000
var synMp = sync.Map{}

func readConMap() {
	for i := 0; i < 10000; i++ {
		key := strconv.Itoa(int(rand.Int63()))
		conMp.Get(key)
	}
}

func writeConMap() {
	for i := 0; i < 10000; i++ {
		key := strconv.Itoa(int(rand.Int63()))
		conMp.Set(key, 1)
	}
}

func writeSynMap() {
	for i := 0; i < 10000; i++ {
		key := strconv.Itoa(int(rand.Int63()))
		synMp.Load(key)
	}
}

func readSynMap() {
	for i := 0; i < 10000; i++ {
		key := strconv.Itoa(int(rand.Int63()))
		synMp.Store(key, 1)
	}
}

func BenchmarkConMap(b *testing.B){
	for i:=0;i<b.N;i++{
		const P = 300
		wg := sync.WaitGroup{}
		wg.Add(2*P)

		for i:=0;i<P;i++{
			go func(){
				defer wg.Done()
				readConMap()
			}()
		}

		for i:=0;i<P;i++{
			go func(){
				defer wg.Done()
				writeConMap()
			}()
		}
	}
}

func BenchmarkSynMap(b *testing.B){
	for i:=0;i < b.N;i++{
		const P = 300
		wg := sync.WaitGroup{}
		wg.Add(2 * P)
		for i:=0;i<P;i++{
			go func(){
				defer wg.Done()
				writeSynMap()
			}()
		}

		for i:=0;i<P;i++{
			go func(){
				defer wg.Done()
				readSynMap()
			}()
		}
	}
}

//go test ./util/test -bench=Map -run=^$ -count=1 -benchmem -benchtime=3s


