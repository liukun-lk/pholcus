// 同时输出报告到子节点。
package reporter

import (
	"fmt"
	"github.com/henrylee2cn/pholcus/config"
	"github.com/henrylee2cn/pholcus/pholcus/node"
	"log"
)

type Report struct {
	status int
}

var Log Reporter

func init() {
	Log = &Report{}
}

func (self *Report) send(str string) {
	if node.Self.GetRunMode() != config.OFFLINE {
		go func() {
			node.Self.NewDataSend(config.LOG, str, "")
		}()
	}
}

func (self *Report) Printf(format string, v ...interface{}) {
	if self.status == config.STOP {
		return
	}
	log.Printf(format, v...)
	self.send(fmt.Sprintf(format, v...))
}

func (self *Report) Println(v ...interface{}) {
	if self.status == config.STOP {
		return
	}
	log.Println(v...)
	self.send(fmt.Sprintln(v...))
}

func (self *Report) Fatal(v ...interface{}) {
	if self.status == config.STOP {
		return
	}
	self.send(fmt.Sprintln(v...))
	log.Fatal(v...)
}

func (self *Report) Stop() {
	self.status = config.STOP
}

func (self *Report) Run() {
	self.status = config.RUN
}
