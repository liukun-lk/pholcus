package gui

import (
	"github.com/henrylee2cn/pholcus/config"
	// "github.com/henrylee2cn/pholcus/pholcus"
	"github.com/henrylee2cn/pholcus/pholcus/crawler"
	"github.com/henrylee2cn/pholcus/reporter"
	"github.com/henrylee2cn/pholcus/scheduler"
	_ "github.com/henrylee2cn/pholcus/spiders"
	"github.com/henrylee2cn/pholcus/spiders/spider"
	"github.com/lxn/walk"
	"log"
	"strconv"
	"time"
)

var (
	toggleSpecialModePB *walk.PushButton
	setting             *walk.Composite
	mw                  *walk.MainWindow
	runMode             *walk.GroupBox
	db                  *walk.DataBinder
	ep                  walk.ErrorPresenter
	mode                *walk.GroupBox
	host                *walk.Splitter
	spiderMenu          = NewSpiderMenu(spider.Menu)
	status              int
)

func Run() {
	runmodeWindow()
}

func writeConf1() {
	config.Task.RunMode = Input.RunMode // 节点角色
	config.Task.Port = Input.Port       // 主节点端口
	config.Task.Master = Input.Master   //服务器(主节点)地址，不含端口
}

func writeConf2() {
	// 纠正协程数
	if Input.ThreadNum == 0 {
		Input.ThreadNum = 1
	}
	config.Task.ThreadNum = Input.ThreadNum
	config.Task.BaseSleeptime = Input.BaseSleeptime
	config.Task.RandomSleepPeriod = Input.RandomSleepPeriod //随机暂停最大增益时长
	config.Task.OutType = Input.OutType
	config.Task.DockerCap = Input.DockerCap //分段转储容器容量
	// 选填项
	config.Task.MaxPage = Input.MaxPage
	config.AutoDockerQueueCap()
}

// 根据GUI提交信息生成蜘蛛列表
func initSpiders() int {
	spider.List.Init()

	// 遍历任务
	for _, sps := range Input.Spiders {
		sps.Spider.SetPausetime(Input.BaseSleeptime, Input.RandomSleepPeriod)
		sps.Spider.SetMaxPage(Input.MaxPage)
		spider.List.Add(sps.Spider)
	}

	// 遍历关键词
	spider.List.ReSetByKeywords(Input.Keywords)

	return spider.List.Len()
}

// 开始执行任务
func Exec(count int) {

	config.ReqSum = 0

	// 初始化资源队列
	scheduler.Init(Input.ThreadNum)

	// 初始化爬虫队列
	CrawlerNum := config.CRAWLER_CAP
	if count < config.CRAWLER_CAP {
		CrawlerNum = count
	}
	crawler.CQ.Init(uint(CrawlerNum))

	log.Println(` ********************************************************************************************************************************************** `)
	log.Printf(" * ")
	log.Printf(" *     执行任务总数（任务数[*关键词数]）为 %v 个...\n", count)
	log.Printf(" *     爬虫队列可容纳蜘蛛 %v 只...\n", CrawlerNum)
	log.Printf(" *     并发协程最多 %v 个……\n", Input.ThreadNum)
	log.Printf(" *     随机停顿时间为 %v~%v ms ……\n", Input.BaseSleeptime, Input.BaseSleeptime+Input.RandomSleepPeriod)
	log.Printf(" * ")
	log.Printf(" *                                                                                                             —— 开始抓取，请耐心等候 ——")
	log.Printf(" * ")
	log.Println(` ********************************************************************************************************************************************** `)

	// 开始计时
	config.StartTime = time.Now()

	// 任务执行
	status = config.RUN
	go GoRun(count)
}

// 任务执行
func GoRun(count int) {
	for i := 0; i < count && status == config.RUN; i++ {
		// 从爬行队列取出空闲蜘蛛，并发执行
		c := crawler.CQ.Use()

		if c != nil {
			go func(i int, c crawler.Crawler) {
				// 执行并返回结果消息
				c.Init(spider.List.Get(i)).Start()
				// 任务结束后回收该蜘蛛
				crawler.CQ.Free(c.GetId())
			}(i, c)
		}
	}

	// 监控结束任务
	sum := 0 //数据总数
	for i := 0; i < count; i++ {
		s := <-config.ReportChan

		log.Println(` ********************************************************************************************************************************************** `)
		log.Printf(" * ")
		reporter.Log.Printf(" *     [结束报告 -> 任务：%v | 关键词：%v]   共输出数据 %v 条，用时 %v 分钟！\n", s.SpiderName, s.Keyword, s.Num, s.Time)
		log.Printf(" * ")
		log.Println(` ********************************************************************************************************************************************** `)

		if slen, err := strconv.Atoi(s.Num); err == nil {
			sum += slen
		}
	}

	// 总耗时
	takeTime := time.Since(config.StartTime).Minutes()

	// 打印总结报告
	log.Println(` ********************************************************************************************************************************************** `)
	log.Printf(" * ")
	reporter.Log.Printf(" *                               —— 本次抓取合计 %v 条数据，下载页面 %v 个，耗时：%.5f 分钟 ——", sum, config.ReqSum, takeTime)
	log.Printf(" * ")
	log.Println(` ********************************************************************************************************************************************** `)

	if config.Task.RunMode == config.OFFLINE {
		// 按钮状态控制
		toggleSpecialModePB.SetEnabled(true)
		toggleSpecialModePB.SetText("开始运行")
	}
}

//中途终止任务
func Stop() {
	status = config.STOP
	crawler.CQ.Stop()
	scheduler.Sdl.Stop()
	reporter.Log.Stop()

	// 总耗时
	takeTime := time.Since(config.StartTime).Minutes()

	// 打印总结报告
	log.Println(` ********************************************************************************************************************************************** `)
	log.Printf(" * ")
	log.Printf(" *                               ！！任务取消：下载页面 %v 个，耗时：%.5f 分钟！！", config.ReqSum, takeTime)
	log.Printf(" * ")
	log.Println(` ********************************************************************************************************************************************** `)

	// 按钮状态控制
	toggleSpecialModePB.SetEnabled(true)
	toggleSpecialModePB.SetText("开始运行")
}
