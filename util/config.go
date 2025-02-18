package util

import (
	"path"
	"runtime"
)

var (
	RootPath string //项目根目录
)

func init() {
	RootPath = path.Dir(GetCurrentPath()+"..") + "/"
}
func GetCurrentPath() string {
	_, filename, _, _ := runtime.Caller(1) //1表示当前函数，2表示调用本函数的函数，3...依次类推
	return path.Dir(filename)
}
