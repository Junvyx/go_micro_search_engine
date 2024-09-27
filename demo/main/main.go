package main

import (
	"flag"
	"net/http"
	"search_engine/demo/handler"
	"search_engine/internal/kvdb"
	"search_engine/util"
	"strconv"

	"github.com/gin-gonic/gin"
)

var (
	mode         = flag.Int("mode", 1, "启动哪类服务。1-standalone web server, 2-grpc index server, 3-distributed web server")
	rebuildIndex = flag.Bool("index", false, "server启动时是否需要重建索引")
	port         = flag.Int("port", 0, "server工作端口")
	dbPath       = flag.String("dbPath", "", "正派索引数据的存放路径")
	totalWorkers = flag.Int("totalWorkers", 0, "分布式环境中一共有几台index worker")
	workerIndex  = flag.Int("workerIndex", 0, "本机是第几台index worker(从0开始编号)")
)

var (
	dbType      = kvdb.REDIS                              //正排索引使用哪种KV数据库
	csvFile     = util.RootPath + "data/bili_video_s.csv" //原始的数据文件，由它来创建索引
	etcdServers = []string{"127.0.0.1:2379"}
)

func StartGin() {
	engine := gin.Default()
	gin.SetMode(gin.ReleaseMode)

	engine.Static("js", "demo/views/js")
	engine.Static("css", "demo/views/css")
	engine.Static("img", "demo/views/img")
	engine.StaticFile("/favicon.ico", "img/dqq.png") //在url中访问文件/favicon.ico，相当于访问文件系统中的views/img/dqq.png文件
	engine.LoadHTMLFiles("demo/views/search.html")   //使用这些.html文件时就不需要加路径了

	engine.Use(handler.GetUserInfo)
	//数组，非切片。数组更加稳定且更节省资源。
	classes := [...]string{"咨询", "社会", "热点", "生活", "知识", "环球", "游戏", "综合", "日常", "影视", "科技", "编程"}
	engine.GET("/", func(ctx *gin.Context) {
		ctx.HTML(http.StatusOK, "search.html", classes)
	})

	engine.POST("/search", handler.Search)
	engine.Run("127.0.0.1:" + strconv.Itoa(*port))
}

func main() {
	flag.Parse()

	switch *mode {
	case 1, 3:
		WebServerMain(*mode) //1：单机模式，索引功能嵌套在web server内部。3：分布式模式，web server内持有一个哨兵，通过哨兵去访问各个grpc index server
		StartGin()
	case 2:
		GrpcIndexerMain() //以grpc server的方式启动索引服务
	}
}

// go run ./demo/main -mode=1 -index=true -port=5678 -dbPath=data/local_db/video_redis
// go run ./demo/main -mode=2 -index=true -port=5600 -dbPath=data/local_db/video_bolt -totalWorkers=2 -workerIndex=0
// go run ./demo/main -mode=2 -index=true -port=5601 -dbPath=data/local_db/video_bolt -totalWorkers=2 -workerIndex=1
// go run ./demo/main -mode=3 -index=true -port=5678
