package main

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"search_engine/demo"
	indexservice "search_engine/index_service"
	"search_engine/util"
	"strconv"
	"syscall"

	"google.golang.org/grpc"
)

var service *indexservice.IndexServiceWorker //IndexWorker，是一个grpc server

func GrpcIndexerInit() {
	//监听本地端口
	lis, err := net.Listen("tcp", "127.0.0.1:"+strconv.Itoa(*port))
	if err != nil {
		panic(err)
	}

	server := grpc.NewServer()
	service = new(indexservice.IndexServiceWorker)
	//初始化索引
	service.Init(50000, dbType, *dbPath+"_part"+strconv.Itoa(*workerIndex))
	if *rebuildIndex {
		util.Log.Printf("totalWorkers=%d, workerIndex=%d", *totalWorkers, *workerIndex)
		demo.BuildIndexFromFile(csvFile, service.Indexer, *totalWorkers, *workerIndex)
	} else {
		service.Indexer.LoadFromIndexFile() //直接从正排索引文件里加载
	}
	//注册服务的具体实现
	indexservice.RegisterIndexServiceServer(server, service)

	//启动服务
	fmt.Printf("start grpc server on port %d\n", *port)
	//向注册中心注册自己，并周期性续命
	service.Regist(etcdServers, *port) //这里的port是命令行里指定的
	err = server.Serve(lis)            //Serve会一直阻塞，所以放到一个协程里异步执行
	if err != nil {
		service.Close()
		fmt.Printf("start grpc server on port %d failed:%s\n", *port, err)
	}
}

func GrpcIndexerTeardown() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	service.Close() //接收到kill信号时关闭索引
	os.Exit(0)      //然后自杀
}

func GrpcIndexerMain() {
	go GrpcIndexerTeardown()
	GrpcIndexerInit()
}
