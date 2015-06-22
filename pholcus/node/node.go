package node

import (
	"encoding/json"
	"github.com/henrylee2cn/pholcus/config"
	"github.com/henrylee2cn/pholcus/pholcus/task"
	"log"
	"net"
	"strconv"
	"time"
)

type Node struct {
	localAddr string
	runMode   int
	port      string
	master    string
	nodes     map[string]*Conn
	*task.TaskJar
	ReceiveDocker chan *Data // 接收数据的缓存池(目前客户端存放来自服务器的Tasks，服务器存放来自客户端的报告)
	SendDocker    chan *Data // 发送数据的缓存池(目前只在客户端存放将要发送给服务器的报告)
}

func New() *Node {
	return &Node{
		runMode:       config.Task.RunMode,
		port:          ":" + strconv.Itoa(config.Task.Port),
		master:        config.Task.Master,
		nodes:         map[string]*Conn{},
		TaskJar:       task.NewTaskJar(),
		ReceiveDocker: make(chan *Data, config.Task.DockerCap),
		SendDocker:    make(chan *Data, config.Task.DockerCap),
	}
}

// 声明实例
var Self *Node = nil

// 运行节点
func RunSelf() {
	if Self != nil {
		return
	}
	Self = New()
	switch Self.runMode {
	case config.SERVER:
		if Self.checkPort() {
			log.Printf("                                                                                                          ！！当前运行模式为：[ 服务器 ] 模式！！")
			go Self.server()
		}

	case config.CLIENT:
		if Self.checkAll() {
			log.Printf("                                                                                                          ！！当前运行模式为：[ 客户端 ] 模式！！")
			go Self.client()
		}
	// case config.OFFLINE:
	// 	fallthrough
	default:
		log.Printf("                                                                                                          ！！当前运行模式为：[ 单机 ] 模式！！")
		return
	}
}

// 生成task并添加至库，服务器模式专用
func (self *Node) AddNewTask(spiders []string, keywords string) {
	task := new(task.Task)

	task.Spiders = spiders
	task.Keywords = keywords

	// 从配置读取字段
	task.ThreadNum = config.Task.ThreadNum
	task.BaseSleeptime = config.Task.BaseSleeptime
	task.RandomSleepPeriod = config.Task.RandomSleepPeriod
	task.OutType = config.Task.OutType
	task.DockerCap = config.Task.DockerCap
	task.DockerQueueCap = config.Task.DockerQueueCap
	task.MaxPage = config.Task.MaxPage

	// 存入
	self.TaskJar.Push(task)
	log.Printf(" *     [新增任务]   详情： %#v", *task)
}

// 请求任务，客户端模式专用
func (self *Node) DownTask() *task.Task {
	if len(self.TaskJar.Ready) == 0 {
		go self.NewDataSend(config.REQTASK, nil, "")
	}

	for len(self.TaskJar.Ready) == 0 {
		time.Sleep(5e7)
	}
	return self.TaskJar.Pull()
}

// 轮询等待，直到有连接生成
func (self *Node) WaitConn() {
	for len(self.nodes) == 0 {
		time.Sleep(5e8)
	}
}

func (self *Node) GetRunMode() int {
	return self.runMode
}

// 生成并发送信息，注意body不可为变量地址
func (self *Node) NewDataSend(Type int, Body interface{}, To string) {
	self.SendDocker <- &Data{
		Type: Type,
		Body: Body,
		To:   To,
	}
}

func (self *Node) server() {
	listener, err := net.Listen("tcp", self.port)
	checkError(err)

	log.Println(" *     —— 已开启服务器监听 ——")
	for {
		// 等待下一个连接,如果没有连接,listener.Accept会阻塞
		conn, err := listener.Accept()
		if err != nil {
			continue
		}

		conn.SetReadDeadline(time.Now().Add(2 << 10 * time.Hour)) // set timeout

		// 登记连接
		go self.autoHandle(conn)

		log.Printf(" *     —— 客户端 %v 连接成功 ——", conn.RemoteAddr().String())
	}
}

func (self *Node) client() {
	log.Println(" *     —— 正在连接服务器……")

RetryLabel:
	conn, err := net.Dial("tcp", config.Task.Master+self.port)
	if err != nil {
		time.Sleep(1e9)
		goto RetryLabel
	}
	conn.SetReadDeadline(time.Now().Add(2 << 10 * time.Hour)) // set timeout

	// 登记连接
	go self.autoHandle(conn)

	log.Printf(" *     —— 成功连接到服务器：%v ——", conn.RemoteAddr().String())
}

func (self *Node) autoHandle(conn net.Conn) {
	if self.localAddr == "" {
		// self.localAddr = strings.Split(conn.LocalAddr().String(), ":")[0]
		self.localAddr = conn.LocalAddr().String()
	}

	// if _, ok := self.nodes[conn.RemoteAddr().String()]; ok {
	// 	return
	// }
	c := NewConn(conn)
	self.nodes[c.RemoteAddr()] = c

	// 开启处理协程
	switch self.runMode {
	case config.SERVER:
		self.serverHandle(c)
	case config.CLIENT:
		self.clientHandle(c)
	}
}

// 服务器先读后写
func (self *Node) serverHandle(conn *Conn) {
	request := make([]byte, 4096) // set maxium request length to 2048KB to prevent flood attack
	defer func() {
		conn.Close()
		// close connection before exit
		Self = New()
	}()
	for {
		read_len, err := conn.Read(request)
		if err != nil {
			log.Println(err)
			break
		}

		if read_len == 0 {
			break // connection already closed by client
		}

		data, err := self.unmarshal(request[:read_len])
		if err != nil {
			break
		}

		self.serverReceive(data)

		request = make([]byte, 4096) // clear last read content
	}
}

// 处理接收的数据
func (self *Node) serverReceive(data *Data) {
	log.Println("接收到", *data)
	switch data.Type {
	case config.REQTASK:
		self.dealReqTask(data)
	case config.LOG:
		go self.dealLog(data)
	default:
		go self.dealLog(data)
	}
}

// 客户端先写后读
func (self *Node) clientHandle(conn *Conn) {
	defer func() {
		conn.Close()
		// close connection before exit
		Self = New()
	}()
	request := make([]byte, 4096) // set maxium request length to 2048KB to prevent flood attack
	i := 0
	for {
		i++
		log.Printf("第 %v 次写入请求", i)
		log.Printf("发送通道剩余数据 %v 个", len(self.SendDocker))

		gotoRead := self.clientSend()
		log.Println("gotoRead:", gotoRead)
		if gotoRead {

			read_len, err := conn.Read(request)
			if err != nil {
				log.Println(err)
				break
			}

			if read_len == 0 {
				break // connection already closed by client
			}

			d, err := self.unmarshal(request[:read_len])
			if err != nil {
				break
			}

			self.clinetReceive(d)
			request = make([]byte, 4096) // clear last read content
		}
	}
}

func (self *Node) clientSend() (gotoRead bool) {
	data := <-self.SendDocker
	log.Println("取出数据", data)

	switch data.Type {
	case config.REQTASK:
		gotoRead = true
	case config.LOG:
		gotoRead = false
	}

	self.autoSend(data)

	return
}

// 处理接收的数据
func (self *Node) clinetReceive(data *Data) {
	log.Println("接收到", *data)
	switch data.Type {
	case config.TASK:
		go self.dealTask(data)
	default:
		go self.dealLog(data)
	}
}

// 分发任务
func (self *Node) dealReqTask(data *Data) {
	var t task.Task
	var ok bool
	for {
		if t, ok = self.TaskJar.Out(data.From, len(self.nodes)); ok {
			break
		}
		time.Sleep(1e9)
	}
	self.NewDataSend(config.TASK, t, data.From)
	self.autoSend(<-self.SendDocker)
}

// 将接收来的任务加入库
func (self *Node) dealTask(data *Data) {
	log.Println("将任务入库", data)
	d, err := json.Marshal(data.Body)
	if err != nil {
		log.Println("json编码失败", data.Body)
		return
	}
	t := &task.Task{}
	err = json.Unmarshal(d, t)
	if err != nil {
		log.Println("json解码失败", data.Body)
		return
	}
	self.TaskJar.Into(t)
}

// 打印报告
func (self *Node) dealLog(data *Data) {
	log.Println(` ********************************************************************************************************************************************** `)
	log.Printf(" * ")
	log.Printf(" *     客户端 [ %s ]    %v", data.From, data.Body)
	log.Printf(" * ")
	log.Println(` ********************************************************************************************************************************************** `)
}
func (self *Node) checkPort() bool {
	if config.Task.Port == 0 {
		log.Println(" *     —— 亲，分布式端口不能为空哦~")
		return false
	}
	return true
}

//实时发送点对点信息
func (self *Node) autoSend(data *Data) {

	if data.To == "" {
		self.randomSend(data)
	} else {
		self.send(self.nodes[data.To], data)
	}
}

// 随机点对点发信息
func (self *Node) randomSend(data *Data) {
	self.WaitConn()
	for _, conn := range self.nodes {
		self.send(conn, data)
		return
	}
}

func (self *Node) sendWithClose(conn *Conn, data *Data) {
	self.send(conn, data)
	conn.Close()
	delete(self.nodes, conn.RemoteAddr())
}

func (self *Node) send(conn *Conn, data *Data) {
	data.From = self.localAddr
	d, err := self.marshal(data)
	if err != nil {
		log.Println("编码出错了", err)
		return
	}
	conn.Write(d)
	log.Println("信息已发送", data)
}

func (self *Node) checkAll() bool {
	if config.Task.Master == "" || !self.checkPort() {
		log.Println(" *     —— 亲，服务器地址不能为空哦~")
		return false
	}
	return true
}

func checkError(err error) {
	if err != nil {
		log.Printf("Fatal error: %s", err.Error())
	}
}

//编码通信数据
func (self *Node) marshal(data *Data) ([]byte, error) {
	b, err := json.Marshal(*data)
	//[]byte("}\r\n")==[]byte{125,13,10}
	// b = append(b, []byte{13, 10}...)
	return b, err
}

//解码通信数据
func (self *Node) unmarshal(data []byte) (*Data, error) {
	//[]byte("}\r\n")==[]byte{125,13,10}
	// for k, v := range data {
	// 	if v == byte(10) && data[k-1] == byte(13) && data[k-2] == byte(125) {
	// 		data = data[:k]
	// 		break
	// 	}
	// }
	d := new(Data)
	err := json.Unmarshal(data, d)
	return d, err
}
