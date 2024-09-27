package course

//一个文件包含一个Id和若干个关键词
type Doc struct {
	Id       int
	Keywords []string
}

//倒排索引
//关键词对应Id(map)
//不同关键词可能会对应多个Id
func BuildInvertIndex(docs []*Doc) map[string][]int {
	index := make(map[string][]int, 100) //提前预留好空间
	//遍历每个文件
	for _, doc := range docs {
		//遍历每个文件的关键词
		for _, keyword := range doc.Keywords {
			index[keyword] = append(index[keyword], doc.Id)
		}
	}
	return index
}
