package config

import (
	"time"
)

//****************************************全局配置*******************************************\\

const (
	//软件名
	APP_NAME = "Pholcus幽灵蛛数据采集_v0.31 （by henrylee2cn）"
	// 蜘蛛池容量
	CRAWLER_CAP = 50

	// 收集器容量
	DATA_CAP = 2 << 14 //65536

	// mongodb数据库服务器
	DB_URL = "127.0.0.1:27017"

	//mongodb数据库名称
	DB_NAME = "temp-collection-tentinet"

	//mongodb数据库集合
	DB_COLLECTION = "news"
)

//**************************************任务运行时公共配置****************************************\\

// 任务运行时公共配置
type TaskConf struct {
	RunMode           int    // 节点角色
	Port              int    // 主节点端口
	Master            string //服务器(主节点)地址，不含端口
	ThreadNum         uint
	BaseSleeptime     uint
	RandomSleepPeriod uint //随机暂停最大增益时长
	OutType           string
	DockerCap         uint //分段转储容器容量
	DockerQueueCap    uint //分段输出池容量，不小于2
	// 选填项
	MaxPage int
}

var Task = &TaskConf{
	RunMode:           OFFLINE,
	Port:              2015,
	Master:            "127.0.0.1",
	ThreadNum:         20,
	BaseSleeptime:     1000,
	RandomSleepPeriod: 3000,
	OutType:           "csv",
	DockerCap:         10000,

	MaxPage: 100,
}

// 根据Task.DockerCap智能调整分段输出池容量Task.DockerQueueCap
func AutoDockerQueueCap() {
	switch {
	case Task.DockerCap <= 10:
		Task.DockerQueueCap = 500
	case Task.DockerCap <= 500:
		Task.DockerQueueCap = 200
	case Task.DockerCap <= 1000:
		Task.DockerQueueCap = 100
	case Task.DockerCap <= 10000:
		Task.DockerQueueCap = 50
	case Task.DockerCap <= 100000:
		Task.DockerQueueCap = 10
	default:
		Task.DockerQueueCap = 4
	}
}

//****************************************GUI内容配置*******************************************\\

// 下拉菜单辅助结构体
type KV struct {
	Key    string
	Int    int
	Uint   uint
	String string
}

// 暂停时间选项及输出类型选项
var GuiOpt = struct {
	OutType   []*KV
	SleepTime []*KV
	RunMode   []*KV
}{
	OutType: []*KV{
		{Key: "csv", String: "csv"},
		{Key: "excel", String: "excel"},
		{Key: "mongoDB", String: "mongoDB"},
	},
	SleepTime: []*KV{
		{Key: "无暂停", Uint: 0},
		{Key: "0.1 秒", Uint: 100},
		{Key: "0.3 秒", Uint: 300},
		{Key: "0.5 秒", Uint: 500},
		{Key: "1 秒", Uint: 1000},
		{Key: "3 秒", Uint: 3000},
		{Key: "5 秒", Uint: 5000},
		{Key: "10 秒", Uint: 10000},
		{Key: "15 秒", Uint: 15000},
		{Key: "20 秒", Uint: 20000},
		{Key: "30 秒", Uint: 30000},
		{Key: "60 秒", Uint: 60000},
	},
	RunMode: []*KV{
		{Key: "单机", Int: OFFLINE},
		{Key: "服务器", Int: SERVER},
		{Key: "客户端", Int: CLIENT},
	},
}

//****************************************任务报告*******************************************\\

type Report struct {
	SpiderName string
	Keyword    string
	Num        string
	Time       string
}

var (
	// 点击开始按钮的时间点
	StartTime time.Time
	// 小结报告通道
	ReportChan chan *Report
	// 请求页面计数
	ReqSum uint
)

//****************************************节点配置*******************************************\\

// 运行模式
const (
	OFFLINE = iota
	SERVER
	CLIENT
)

// 数据头部信息
const (
	// 任务请求Header
	REQTASK = iota + 1
	// 任务响应流头Header
	TASK
	// 打印Header
	LOG
)

//****************************************其他常量*******************************************\\

// 运行状态
const (
	STOP = 0
	RUN  = 1
)

//****************************************相关初始化*******************************************\\

func init() {
	// 任务报告
	ReportChan = make(chan *Report)

	// 根据Task.DockerCap智能调整分段输出池容量Task.DockerQueueCap
	AutoDockerQueueCap()
}
