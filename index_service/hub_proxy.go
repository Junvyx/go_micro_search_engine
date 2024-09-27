package indexservice

import (
	"context"
	"search_engine/util"
	"strings"
	"sync"
	"time"

	etcdv3 "go.etcd.io/etcd/client/v3"
	"golang.org/x/time/rate"
)

// 代理和直接访问都是用了这些方法
type IServiceHub interface {
	Regist(service string, endpoint string, leaseID etcdv3.LeaseID) (etcdv3.LeaseID, error) // 注册服务
	UnRegist(service string, endpoint string) error                                         // 注销服务
	GetServiceEndpoints(service string) []string                                            //服务发现
	GetServiceEndpoint(service string) string                                               //选择服务的一台endpoint
	Close()                                                                                 //关闭etcd client connection
}

// 代理模式
// 对ServiceHub做一层代理，想访问endpoints时需要通过代理，代理提供了2个功能：缓存和限流保护
type HubProxy struct {
	*ServiceHub             //匿名成员，后面要用成员名就是类型名ServiceHub（不带*）。这样可以直接使用你名成员的方法
	endpointsCache sync.Map //维护每一个service下的所有servers
	limiter        *rate.Limiter
}

var (
	proxy     *HubProxy
	proxyOnce sync.Once
)

// HubProxy的构造函数，单例模式。
//
// qps一秒钟最多允许请求多少次
func GetServiceHubProxy(etcdServers []string, heartbeatFrequency int64, qps int) *HubProxy {
	if proxy == nil {
		proxyOnce.Do(func() {
			servicehub := GetServiceHub(etcdServers, heartbeatFrequency)
			if servicehub != nil {
				proxy = &HubProxy{
					ServiceHub:     servicehub,
					endpointsCache: sync.Map{},
					limiter:        rate.NewLimiter(rate.Every(time.Duration(1e9/qps)*time.Nanosecond), qps),
					//每隔1E9/qps纳秒产生一个令牌，即一纳秒钟之内产生qps个令牌。令牌桶的容量为qps
				}
			}
		})
	}
	return proxy
}

// 监听服务是否还存在
// 并将serive和endpoints存入endpointsCache
func (proxy *HubProxy) watchEndpointsOfService(service string) {
	//监听了就赋值true？？
	if _, exists := proxy.watched.LoadOrStore(service, true); exists { //watched是从父类继承过来的
		return //监听过了，不用重复监听
	}
	ctx := context.Background()
	prefix := strings.TrimRight(SERVICE_ROOT_PATH, "/") + "/" + service + "/"
	ch := proxy.client.Watch(ctx, prefix, etcdv3.WithPrefix()) //返回的是管道
	//根据前缀监听，每一个修改都会放入管道ch。client是从父类继承过来的
	//ch里面存的是etcdv3.WatchResponse
	util.Log.Printf("监听服务%s的节点变化", service)
	go func() {
		for response := range ch { //遍历管道。这是个死循环，除非关闭管道()
			for _, event := range response.Events { //[]*etcdv3.Event
				util.Log.Printf("etcd event type %s", event.Type) //PUT或DELETE
				path := strings.Split(string(event.Kv.Key), "/")
				if len(path) > 2 {
					service := path[len(path)-2] //倒一是ip，倒二才是服务
					// 跟etcd进行一次全量同步
					endpoints := proxy.ServiceHub.GetServiceEndpoints(service)
					//显式调用父类的GetServiceEndpoints()
					//如果没有重写可以隐式调用
					if len(endpoints) > 0 {
						proxy.endpointsCache.Store(service, endpoints)
					} else {
						proxy.endpointsCache.Delete(service)
					}
				}
			}
		}
	}()
}

func (proxy *HubProxy) GetServiceEndpoints(service string) []string {
	if !proxy.limiter.Allow() { //不阻塞，如果桶中没有1个令牌，则函数直接返回空，即没有可用的endpoints
		return nil
	}

	proxy.watchEndpointsOfService(service) //监听etcd的数据变化，及时更新本地缓存
	if endpoints, exists := proxy.endpointsCache.Load(service); exists {
		return endpoints.([]string)
	} else {
		//缓存中没有就去服务器找
		endpoints := proxy.ServiceHub.GetServiceEndpoints(service)
		if len(endpoints) > 0 {
			proxy.endpointsCache.Store(service, endpoints)
		}
		return endpoints
	}
}
