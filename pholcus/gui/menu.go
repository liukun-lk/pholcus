package gui

import (
	"github.com/henrylee2cn/pholcus/config"
)

// GUI输入
type Inputor struct {
	Keywords string //后期split()为slice
	Spiders  []*GUISpider
	*config.TaskConf
}

var Input = &Inputor{
	// 默认值
	TaskConf: config.Task,
}
