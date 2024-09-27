package types

import "strings"

// ((A|B|C) & E) | F
// 每个基础元素是一个TermQuery（括号不是）
// 每个元素合在一起（进行若干或与运算）也是TermQuery
type TermQueryV0 struct {
	Must    []TermQueryV0
	Should  []TermQueryV0
	Keyword string
}

func (exp TermQueryV0) Empty() bool {
	return len(exp.Keyword) == 0 && len(exp.Must) == 0 && len(exp.Should) == 0
}

func KeywordExperssion(keyword string) TermQueryV0 {
	return TermQueryV0{Keyword: keyword}
}

func MustExpression(exps ...TermQueryV0) TermQueryV0 {
	if len(exps) == 0 {
		return TermQueryV0{}
	}
	array := make([]TermQueryV0, 0, len(exps))
	//非空的Expression才能添加到array里面去
	for _, exp := range exps {
		if !exp.Empty() {
			array = append(array, exp)
		}
	}
	return TermQueryV0{Must: array}
}

func ShouldExpression(exps ...TermQueryV0) TermQueryV0 {
	if len(exps) == 0 {
		return TermQueryV0{}
	}
	array := make([]TermQueryV0, 0, len(exps))
	for _, exp := range exps {
		if !exp.Empty() {
			array = append(array, exp)
		}
	}
	return TermQueryV0{Should: array}
}

//print函数会自动调用变量的String()方法
func (exp TermQueryV0) String() string {
	if len(exp.Keyword) > 0 {
		return exp.Keyword
	} else if len(exp.Must) > 0 {
		if len(exp.Must) == 1 {
			return exp.Must[0].String()
		} else {
			sb := strings.Builder{}
			sb.WriteByte('(')
			for _, e := range exp.Must {
				sb.WriteString(e.String())
				sb.WriteByte('&')
			}
			s := sb.String()
			s = s[0:len(s)-1] + ")" //删除最后一个'&'？
			return s
		}
	} else if len(exp.Should) > 0 {
		if len(exp.Should) == 1 {
			return exp.Should[0].String()
		} else {
			sb := strings.Builder{}
			sb.WriteByte('(')
			for _, e := range exp.Should {
				sb.WriteString(e.String())
				sb.WriteByte('|')
			}
			s := sb.String()
			s = s[0:len(s)-1] + ")"
			return s
		}
	}
	return ""
}
