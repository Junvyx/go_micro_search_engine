package main

import (
	"os"
	"os/signal"
	"search_engine/demo"
	"search_engine/demo/handler"
	indexservice "search_engine/index_service"
	"syscall"
)

func WebServerInit(mode int) {
	switch mode {
	case 1:
		standaloneIndexer := new(indexservice.Indexer) //单机索引服务启动
		if err := standaloneIndexer.Init(50000, dbType, *dbPath); err != nil {
			panic(err)
		}

		if *rebuildIndex {
			demo.BuildIndexFromFile(csvFile, standaloneIndexer, 0, 0) //重建索引
		} else {
			standaloneIndexer.LoadFromIndexFile() //直接从正排索引文件里加载
		}
		handler.Indexer = standaloneIndexer
	case 3:
		handler.Indexer = indexservice.NewSentinel(etcdServers)
	default:
		panic("invalid mode")
	}
}

func WebServerTeardown() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	handler.Indexer.Close() //接收到kill信号时关闭索引
	os.Exit(0)              //然后自杀
}

func WebServerMain(mode int) {
	go WebServerTeardown()
	WebServerInit(mode)
}
