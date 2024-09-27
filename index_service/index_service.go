package indexservice

import (
	"context"
	fmt "fmt"
	"search_engine/types"
	"search_engine/util"
	"strconv"
	"time"
)

const (
	INDEX_SERVICE = "index_service"
)

// IndexWorker，是一个grpc server
type IndexServiceWorker struct {
	Indexer *Indexer // 把正排和倒排索引
	//跟服务注册相关的配置
	hub      *ServiceHub
	selfAddr string //有些情况会用到本机的ip地址
}

// 初始化索引
func (service *IndexServiceWorker) Init(DocNumEstimate int, dbtype int, DataDir string) error {
	service.Indexer = new(Indexer)
	return service.Indexer.Init(DocNumEstimate, dbtype, DataDir)
}

func (service *IndexServiceWorker) Regist(etcdServers []string, servicePort int) error {
	//向注册中心注册自己
	if len(etcdServers) > 0 {
		if servicePort < 1024 { //1024以下的端口一般是比较著名的端口
			return fmt.Errorf("invalid listen port %d, should more than 1024", servicePort)
		}
		selfLocalIp, err := util.GetLocalIP() //获取本机地址
		if err != nil {
			panic(err)
		}
		//TODO 单机模拟分布式时，把selfLocalIp写死为127.0.0.1
		//非单机模式就删掉
		selfLocalIp = "127.0.0.1"
		service.selfAddr = selfLocalIp + ":" + strconv.Itoa(servicePort)
		var heartBeat int64 = 3
		hub := GetServiceHub(etcdServers, heartBeat)                   //获取etcd集群
		leaseId, err := hub.Regist(INDEX_SERVICE, service.selfAddr, 0) //注册
		if err != nil {
			panic(err)
		}
		service.hub = hub
		//周期性注册自己（上报心跳）
		go func() {
			for {
				hub.Regist(INDEX_SERVICE, service.selfAddr, leaseId)
				time.Sleep(time.Duration(heartBeat)*time.Second - 100*time.Millisecond)
				//注册间隔略小于心跳
			}
		}()
	}
	return nil
}

// 系统重启时，直接从索引文件里加载数据
func (service *IndexServiceWorker) LoadFromIndexFile() int {
	return service.Indexer.LoadFromIndexFile()
}

// 关闭索引，主要是关闭正排索引，把kvdb给关闭掉
func (service *IndexServiceWorker) Close() error {
	if service.hub != nil {
		service.hub.UnRegist(INDEX_SERVICE, service.selfAddr)
	}
	return service.Indexer.Close()
}

// 从索引上删除文档
// 直接利用Indexer的方法
// 再上一层是indexer.go
// 返回值要转成proto文件里写定的格式
func (service *IndexServiceWorker) DeleteDoc(ctx context.Context, docId *DocId) (*AffectedCount, error) {
	return &AffectedCount{int32(service.Indexer.DeleteDoc(docId.DocId))}, nil
}

// 向索引中添加文档(如果已存在，会先删除)
func (service *IndexServiceWorker) AddDoc(ctx context.Context, doc *types.Document) (*AffectedCount, error) {
	n, err := service.Indexer.AddDoc(*doc)
	return &AffectedCount{int32(n)}, err
}

// 检索，返回文档列表
func (service *IndexServiceWorker) Search(ctx context.Context, request *SearchRequest) (*SearchResult, error) {
	result := service.Indexer.Search(request.Query, request.OnFlag, request.OffFlag, request.OrFlags)
	return &SearchResult{Results: result}, nil
}

// 索引里有几个文档
func (service *IndexServiceWorker) Count(ctx context.Context, request *CountRequest) (*AffectedCount, error) {
	return &AffectedCount{int32(service.Indexer.Count())}, nil
}
