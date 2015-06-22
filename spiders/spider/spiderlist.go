package spider

import (
	// "github.com/henrylee2cn/pholcus/config"
	"strings"
)

type Spiders struct {
	list []*Spider
}

var (
	// 任务队列
	List = &Spiders{}

	// GUI菜单列表
	Menu = &Spiders{}
)

func (self *Spiders) Init() {
	self.list = []*Spider{}
}

func (self *Spiders) Add(sp *Spider) {
	sp.Id = self.Len()
	self.list = append(self.list, sp)
}

func (self *Spiders) Len() int {
	return len(self.list)
}

func (self *Spiders) Get(idx int) *Spider {
	return self.list[idx]
}

func (self *Spiders) GetByName(n string) *Spider {
	for _, sp := range self.list {
		if sp.GetName() == n {
			return sp
		}
	}
	return nil
}

func (self *Spiders) GetAll() []*Spider {
	return self.list
}

func (self *Spiders) ReSet(list []*Spider) {
	for i := range list {
		list[i].Id = i
	}
	self.list = list
}

// 专为Keywords拆分新增spider而写，调用此方法前不可为其赋值Keywords
func (self *Spiders) ReSetByKeywords(keywords string) {
	if keywords == "" {
		return
	}

	unit1 := []*Spider{}
	unit2 := []*Spider{}
	for _, v := range self.GetAll() {
		if v.GetKeyword() == "" {
			unit1 = append(unit1, v)
			continue
		}
		unit2 = append(unit2, v)
	}

	self.Init()

	keywordSlice := strings.Split(keywords, "|")
	for _, keyword := range keywordSlice {
		keyword = strings.Trim(keyword, " ")
		if keyword == "" {
			continue
		}
		for _, v := range unit2 {
			v.Keyword = keyword
			c := *v
			self.Add(&c)
		}
	}
	if self.Len() == 0 {
		self.ReSet(append(unit1, unit2...))
	}

	for _, v := range unit1 {
		self.Add(v)
	}
}
