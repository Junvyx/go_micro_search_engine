package indexservice

import (
	"math/rand"
	"sync/atomic"
)

type LoadBalancer interface {
	Take([]string) string //从众多服务器（string）中选出一个服务器
}

// 负载均衡算法--轮询法
type RoundRobin struct {
	acc int64
}

func (b *RoundRobin) Take(endpoints []string) string {
	if len(endpoints) == 0 {
		return ""
	}
	n := atomic.AddInt64(&b.acc, 1)
	index := int(n % int64(len(endpoints)))
	return endpoints[index]
}

// 负载均衡算法--随机法
type RandomSelect struct {
}

func (b *RandomSelect) Take(endpoints []string) string {
	if len(endpoints) == 0 {
		return ""
	}
	index := rand.Intn(len(endpoints)) //[0:len(endpoints))
	return endpoints[index]
}
