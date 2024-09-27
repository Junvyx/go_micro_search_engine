package types

//对protoc命令生成的Keyword.String()方法进行重写
func (kw Keyword) ToString() string {
	if len(kw.Word) > 0 { //有关键词
		return kw.Field + "\001" + kw.Word
	} else {
		return ""
	}
}
