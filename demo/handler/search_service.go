package handler

import (
	"log"
	"net/http"
	"search_engine/demo"
	indexservice "search_engine/index_service"
	"search_engine/types"
	"search_engine/util"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gogo/protobuf/proto"
)

var Indexer indexservice.IIndexer

// 对关键词进行清理，大写转小写，删掉空格
func clearKeywords(words []string) []string {
	keywords := make([]string, 0, len(words))
	for _, w := range words {
		word := strings.TrimSpace(strings.ToLower(w)) //去除空格，转小写
		if len(word) > 0 {
			keywords = append(keywords, word)
		}
	}
	return keywords
}

// 搜索接口
func Search(ctx *gin.Context) {
	var request demo.SearchRequest
	if err := ctx.ShouldBindJSON(&request); err != nil { //前端传来的request字段的信息
		log.Printf("bind request parameter failed: %s", err)
		ctx.String(http.StatusBadRequest, "invalid json")
		return
	}

	keywords := clearKeywords(request.Keywords)
	if len(keywords) == 0 && len(request.Author) == 0 {
		ctx.String(http.StatusBadRequest, "关键词和作者不能同时为空")
		return
	}

	query := new(types.TermQuery)
	if len(keywords) > 0 {
		for _, word := range keywords {
			query = query.And(types.NewTermQuery("content", word)) //满足关键词
		}
	}

	if len(request.Author) > 0 {
		query = query.And(types.NewTermQuery("author", strings.ToLower(request.Author))) //满足作者
	}

	orFlags := []uint64{demo.GetClassBits(keywords)} //满足类别
	docs := Indexer.Search(query, 0, 0, orFlags)
	videos := make([]demo.BilliVideo, 0, len(docs))
	for _, doc := range docs {
		var video demo.BilliVideo
		if err := proto.Unmarshal(doc.Bytes, &video); err == nil { //doc字段（后端）转video字段（业务）
			if video.View >= int32(request.ViewFrom) && (request.ViewTo <= 0 || video.View <= int32(request.ViewTo)) { //满足播放量的区间范围
				videos = append(videos, video)
			}
		}
	}
	util.Log.Printf("return %d videos", len(videos))
	ctx.JSON(http.StatusOK, videos) //把搜索结果以json形式返回给前端
}
