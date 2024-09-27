package kvdb

import (
	"os"
	"search_engine/util"
	"strings"
)

// bolt使用B+树， BADGER使用LSM树
const (
	REDIS = iota
	BOLT
	BADGER
)

// redis也是一种KV数据库，读者可以自行用redis实现IKeyValueDB接口
type IKeyValueDB interface {
	Open() error //初始化DB
	GetDbPath() string
	Set(k, v []byte) error //写入<key,value>
	BatchSet(keys, values [][]byte) error
	Get(k []byte) ([]byte, error)
	BatchGet(keys [][]byte) ([][]byte, error)
	Delete(k []byte) error
	BatchDelete(keys [][]byte) error
	Has(k []byte) bool                       //判断某个key是否存在
	IterDB(fn func(k, v []byte) error) int64 //遍历数据库，返回数据的条数
	IterKey(fn func(k []byte) error) int64   //遍历所有key，返回数据的条数
	Close() error                            //把内存中的数据flush到磁盘，同时释放文件锁
}

// Factory工厂模式，类的创建和使用分开
// 可以创建很多类，但都是用同一个接口
// Get函数就是一个工厂，它返回产品的接口，即它可以返回各种各样的具体产品。
func GetKvDb(dbtype int, path string) (IKeyValueDB, error) {
	paths := strings.Split(path, "/")
	parentPath := strings.Join(paths[0:len(paths)-1], "/") //父路径

	info, err := os.Stat(parentPath)
	if os.IsNotExist(err) { //如果父路径不存在则创建
		util.Log.Printf("create dir %s", parentPath)
		os.MkdirAll(parentPath, os.ModePerm)
	} else { //父路径存在
		if info.Mode().IsRegular() { //如果父路径是个普通文件，则把它删掉
			util.Log.Printf("%s is a regular file, will delete it", parentPath)
			os.Remove(parentPath)
		}
	}

	var db IKeyValueDB
	switch dbtype {
	case BADGER:
		db = new(Badger).WithDataPath(path)
	case BOLT: //默认使用bolt
		db = new(Bolt).WithDataPath(path).WithBucket("mySEngine") //Builder生成器模式
	default:
		db = new(RedisDB).WithDataPath(path) //具体用之前要指定一下db.Addr字段
	}
	err = db.Open() //创建具体KVDB的细节隐藏在Open()函数里。在这里【创建类】
	return db, err
}
