package course

//判断第i位是否为1，i从1开始
func IsBit1(n uint64, i int) bool {
	if i > 64 {
		panic(i)
	}

	c := uint64(1 << (i - 1))

	if n&c == c {
		return true
	} else {
		return false
	}
}

//第i位 置为1，i从1开始
func SetBit1(n uint64, i int) uint64 {
	if i > 64 {
		panic(i)
	}

	c := uint64(1 << (i - 1))

	return n | c
}

//一个整数的二进制里包含几个1
func CountBit1(n uint64) int {
	c := uint64(1) //只有最低位是1
	sum := 0

	for i := 0; i < 64; i++ {
		if n&c == c {
			sum += 1
		}
		c = c << 1
	}
	return sum
}

//
/*
const (
	MALE = 1 << iota
	VIP = 1 << iota
	WEEK_ACTIVE = 1 << iota
)

const (
	MALE = 1 << 0
	VIP = 1 << 1
	WEEK_ACTIVE = 1 << 2
)

三种写法等价
*/

const (
	MALE = 1 << iota
	VIP
	WEEK_ACTIVE
)

type Candidate struct {
	Id     int
	Gender string
	Vip    bool
	Active int    //几天内活跃
	Bits   uint64 //用来判断candidate的状态，是否为vip等等
}

func (c *Candidate) SetMale() {
	c.Gender = "男"
	c.Bits |= MALE
}

func (c *Candidate) SetVip() {
	c.Vip = true
	c.Bits |= VIP
}

func (c *Candidate) SetActive(day int) {
	c.Active = day
	if day <= 7 {
		c.Bits |= WEEK_ACTIVE
	}
}

//判断三个条件是否同时满足
func (c Candidate) Filter1(male, vip, weekActive bool) bool {
	if male && c.Gender != "男" {
		return false
	}
	if vip && !c.Vip {
		return false
	}
	if weekActive && c.Active > 7 {
		return false
	}
	return true
}

//判断N个条件是否同时满足
func (c Candidate) Filter(on uint64) bool {
	return c.Bits&on == on
}

type BitMap struct {
	Table uint64
}

// min已知
func CreateBitMap(min int, arr []int) *BitMap {
	bitMap := new(BitMap)
	for _, ele := range arr {
		n := ele - min
		bitMap.Table = SetBit1(bitMap.Table, n)
	}
	return bitMap
}

//位图求交集
func IntersectionOfBitMap(bm1, bm2 *BitMap, min int) []int {
	rect := make([]int, 0, 100)
	c := bm1.Table & bm2.Table
	for i := 1; i <= 64; i++ {
		if IsBit1(c, i) {
			rect = append(rect, i+min)
		}
	}
	return rect
}

//有序列表求交
func InersectionOfOrderedList(arr, brr []int) []int {
	m, n := len(arr), len(brr)
	if m == 0 || n == 0 {
		return nil
	}

	rect := make([]int, 0, 100)
	var i, j int //自动初始化为0
	for i < m && j < n {
		if arr[i] == brr[j] {
			rect = append(rect, arr[i])
			i++
			j++
		} else if arr[i] > brr[j] {
			j++
		} else {
			i++
		}
	}
	return rect
}
