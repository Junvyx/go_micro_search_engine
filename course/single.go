package course

import (
	"fmt"
	"sync"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// 单例模式
// 在程序运行期间，某个结构体只需要创建一个实例

var single *gorm.DB //通过gorm.Open()创建的gorm.DB是一个连接池，只需要创建它的一个实例
var once sync.Once = sync.Once{}
var lock = &sync.Mutex{}

// 通过上锁的方式只执行一次
func GetDB1() *gorm.DB {
	if single == nil { //只能创建一个实例，因此实例不为空的时候不要操作
		lock.Lock()
		defer lock.Unlock()
		if single == nil {
			single, _ = gorm.Open(mysql.Open(""))
		} else {
			fmt.Println("单例已经创建过了")
		}
	} else {
		fmt.Println("单例已经创建过了")
	}

	return single
}

// 通过init函数只执行一次
// 但使用init()通常要小心代码的各种依赖关系，关心代码的执行顺序
func init() {
	single, _ = gorm.Open(mysql.Open(""))
}

func GetDB2() *gorm.DB {
	return single
}

// 通过sync.Once只执行一次
func GetDB3() *gorm.DB {
	if single == nil {
		once.Do(
			func() {
				single, _ = gorm.Open(mysql.Open(""))
			}) //once.DO这两个括号要在一起
	} else {
		fmt.Println("单例已经创建过了")
	}

	return single
}
