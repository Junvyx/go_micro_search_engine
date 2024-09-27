package dao

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
)

var loc *time.Location //不能在这直接赋值，因为有两个返回参数

const BATCH_SIZE = 300

// 适合使用init()的典型场景：全局变量的初始化放到init()里，且没有任何前提依赖
func init() {
	var err error
	loc, err = time.LoadLocation("Asia/Shanghai")
	if err != nil {
		panic(err)
	}
}

type BilliVideo struct {
	Id        string
	Title     string
	Author    string
	Post_time time.Time
	Views     int
	ThumbsUp  int //结构体里的驼峰转为蛇形，即mysql表里的列名
	Coin      int
	Favorite  int
	Share     int
	Keywords  string
}

// 读出每行后的操作
func parseFileLine(record []string) *BilliVideo {
	video := &BilliVideo{
		Title:  record[1],
		Author: record[3],
	}
	urlPaths := strings.Split(record[0], "/")
	video.Id = urlPaths[len(urlPaths)-1] //bv
	if len(record[2]) > 4 {
		t, err := time.ParseInLocation("2006/1/2 15:4", record[2], loc)
		if err != nil {
			log.Printf("parse time %s failed: %s", record[2], err)
		} else {
			video.Post_time = t
		}
	}

	n, _ := strconv.Atoi(record[4])
	video.Views = n
	n, _ = strconv.Atoi(record[5])
	video.ThumbsUp = n
	n, _ = strconv.Atoi(record[6])
	video.Coin = n
	n, _ = strconv.Atoi(record[7])
	video.Favorite = n
	n, _ = strconv.Atoi(record[8])
	video.Share = n
	video.Keywords = strings.ToLower(record[9]) //keywords记录里是一段双引号里的句子
	return video
}

// 放到通道的理由是一个文件里有很多条记录，如果放到切片那很占内存
// 放到通道可以设置通道的大小，节省内存
func readFile(csvFile string, ch chan<- *BilliVideo) {
	file, err := os.Open(csvFile)
	if err != nil {
		log.Printf("open file %s failed: %s", csvFile, err)
		return
	}
	defer file.Close()

	reader := csv.NewReader(file)
	for {
		record, err := reader.Read() //一次一条，record是个切片
		if err != nil {
			if err != io.EOF {
				log.Printf("read record failed:%s", err)
			}
			break //到文件尾部了
		}
		if len(record) < 10 { //避免数组越界，发生panic
			continue
		}
		video := parseFileLine(record)
		ch <- video
	}
	close(ch)
}

func DumpDataFromFile2DB1(csvFile string) {
	begin := time.Now()
	defer func(begin time.Time) {
		fmt.Printf("DumpDataFromFile2DB1 use time %d ms\n", time.Since(begin).Milliseconds())
	}(begin)

	ch := make(chan *BilliVideo, 200)
	go readFile(csvFile, ch)

	db := GetSearchDBConnection()
	for {
		video, ok := <-ch
		if !ok {
			break
		}
		err := db.Create(video).Error
		checkErr(err)
	}
}

// 将好多个提交包装成一个提交事务，节省资源（之前是提交300次，这次是一次做300次提交）
func DumpDataFromFile2DB2(csvFile string) {
	begin := time.Now()
	defer func(begin time.Time) {
		fmt.Printf("DumpDataFromFile2DB1 use time %d ms\n", time.Since(begin).Milliseconds())
	}(begin)

	ch := make(chan *BilliVideo, 200)
	go readFile(csvFile, ch)

	db := GetSearchDBConnection()
	tx := db.Begin()
	i := 0
	for {
		video, ok := <-ch
		if !ok {
			break
		}
		tx.Create(video) //通过事务提交insert请求
		i++
		if i >= BATCH_SIZE {
			err := tx.Commit().Error //300次insert提交一次事务
			checkErr(err)
			tx = db.Begin() //不能在一个事务上重复commit，需要新开一个事务
			i = 0
		}
	}
	err := tx.Commit().Error
	checkErr(err)
}

// 又和事务不一样了，这是一次性提交300次记录，只提交一次。不是一次进行300次提交
func DumpDataFromFile2DB3(csvFile string) {
	begin := time.Now()
	defer func(begin time.Time) {
		fmt.Printf("DumpDataFromFile2DB3 use time %d ms\n", time.Since(begin).Milliseconds())
	}(begin)

	ch := make(chan *BilliVideo, 200)
	go readFile(csvFile, ch)

	db := GetSearchDBConnection()
	buffer := make([]*BilliVideo, 0, BATCH_SIZE)
	for {
		video, ok := <-ch
		if !ok {
			break
		}
		buffer = append(buffer, video)
		if len(buffer) >= BATCH_SIZE {
			err := db.CreateInBatches(buffer, BATCH_SIZE).Error //300条数据批量insert
			checkErr(err)
			buffer = make([]*BilliVideo, 0, BATCH_SIZE)
		}
	}
	err := db.CreateInBatches(buffer, BATCH_SIZE).Error
	checkErr(err)
}

func checkErr(err error) {
	var sqlErr *mysql.MySQLError
	if errors.As(err, &sqlErr) {
		if sqlErr.Number != 1062 {
			panic(err) //主键重复文件，这里我们允许主键重复
		}
	}
}

// 最简单的读出全表
func ReadAllTable1(ch chan<- BilliVideo) {
	begin := time.Now()
	defer func(begin time.Time) {
		fmt.Printf("ReadAllTable1 use time %d ms\n", time.Since(begin).Milliseconds())
	}(begin)

	db := GetSearchDBConnection()
	var datas []BilliVideo
	// select * from bili_video; 绝对禁止这种写法，绝对是慢查询
	if err := db.Select("*").Find(&datas).Error; err != nil {
		log.Printf("ReadAllTable1 failed:%s", err)
	}
	for _, data := range datas {
		ch <- data
	}
	log.Printf("ReadAllTable1 read %d records", len(datas))
	close(ch)
}

// 普通的分页查询遍历全表
func ReadAllTable2(ch chan<- BilliVideo) {
	begin := time.Now()
	defer func(begin time.Time) {
		fmt.Printf("ReadAllTable2 use time %d ms \n", time.Since(begin).Milliseconds())
	}(begin)

	db := GetSearchDBConnection()
	offset := 0
	const BATCH = 500
	for {
		t0 := time.Now()
		var datas []BilliVideo
		// select * from bili_video limit offset,BATCH; 实际上执行的是 limit 0,offset+BATCH, 然后截取了最后BATCH个，所以offset越大执行得越慢
		if err := db.Select("*").Offset(offset).Limit(BATCH).Find(&datas).Error; err != nil {
			log.Printf("ReadAllTable2 failed: %s", err)
		} else {
			if len(datas) == 0 {
				break
			}
			for _, data := range datas {
				ch <- data
			}
			offset += len(datas)
		}
		//计算一页花了多少时间
		fmt.Printf("offset=%d use time %dms\n", offset, time.Since(t0).Milliseconds())
	}
	log.Printf("ReadAllTable2 read %d records", offset)
	close(ch)
}

// 借助于主键的有序性，分区段遍历全表
// 这样就不用每次把offset+BATCH前的都查了
func ReadAllTable3(ch chan<- BilliVideo) {
	begin := time.Now()
	defer func(begin time.Time) {
		fmt.Printf("ReadAllTable3 use time %d ms\n", time.Since(begin).Milliseconds())
	}(begin)

	db := GetSearchDBConnection()
	maxid := ""
	const BATCH = 500
	total := 0
	for {
		t0 := time.Now()
		var datas []BilliVideo
		// select * from bili_video where id>maxid limit BATCH; 默认自带 order by id
		if err := db.Select("*").Where("id>?", maxid).Limit(BATCH).Find(&datas).Error; err != nil {
			log.Printf("ReadAllTable3 failed: %s", err)
			break
		} else {
			if len(datas) == 0 {
				break
			}
			for _, data := range datas {
				ch <- data
			}
			maxid = datas[len(datas)-1].Id //最后一个元素的id是最大的(自增)
			total += len(datas)            //记录加载了多少条通道里新增了多少条数据
		}
		fmt.Printf("progress=%d use time %dms\n", total, time.Since(t0))
	}
	log.Printf("ReadAllTable3 read %d records", total)
	close(ch)
}
