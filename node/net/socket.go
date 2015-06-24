// socket通信
package net

import (
	"github.com/henrylee2cn/pholcus/runtime/cache"
	"github.com/henrylee2cn/pholcus/runtime/status"
	"log"
	"net"
	"time"
)

type Network struct {
	// 运行模式
	RunMode int
	// 服务器端口号
	Port string
	// 服务器地址（不含Port）
	Master string
	// 本节点地址（含Port）
	LocalAddr string
	// 连接池
	Conns map[string]*Conn
}

//****************************************服务器*******************************************\\

func (self *Network) Server() {
	listener, err := net.Listen("tcp", self.Port)
	checkError(err)

	log.Println(" *     —— 已开启服务器监听 ——")
	for {
		// 等待下一个连接,如果没有连接,listener.Accept会阻塞
		conn, err := listener.Accept()
		if err != nil {
			continue
		}

		conn.SetReadDeadline(time.Now().Add(2 << 10 * time.Hour)) // set timeout

		// 开启该连接处理协程
		go self.serverHandle(self.perHandle(conn))

		log.Printf(" *     —— 客户端 %v 连接成功 ——", conn.RemoteAddr().String())
	}
}

// 服务器先读后写
func (self *Network) serverHandle(conn *Conn) {
	request := make([]byte, 4096) // set maxium request length to 2048KB to prevent flood attack
	defer func() {
		conn.Close()
	}()
	for {
		read_len, err := conn.Read(request)
		if err != nil {
			log.Println(err)
			continue
		}

		if read_len == 0 {
			continue // connection already closed by client
		}

		data, err := unmarshal(request[:read_len])
		if err != nil {
			continue
		}

		self.serveReceive(data)

		request = make([]byte, 4096) // clear last read content
	}
}

// 处理接收数据
func (self *Network) serveReceive(data *cache.NetData) {
	log.Println("接收到", *data)
	switch data.Type {
	case status.REQTASK:
		cache.ReceiveDocker <- data
		self.GetConn(data.From).Block()
	case status.LOG:
		self.log(data)
	default:
		self.log(data)
	}
}

//****************************************客户端*******************************************\\

func (self *Network) Client() {
	log.Println(" *     —— 正在连接服务器……")

RetryLabel:
	conn, err := net.Dial("tcp", cache.Task.Master+self.Port)
	if err != nil {
		time.Sleep(1e9)
		goto RetryLabel
	}
	conn.SetReadDeadline(time.Now().Add(2 << 10 * time.Hour)) // set timeout

	// 开启该连接处理协程
	go self.clientHandle(self.perHandle(conn))

	log.Printf(" *     —— 成功连接到服务器：%v ——", conn.RemoteAddr().String())
}

// 客户端先写后读
func (self *Network) clientHandle(conn *Conn) {
	defer func() {
		conn.Close()
		// close connection before exit
	}()
	request := make([]byte, 4096) // set maxium request length to 2048KB to prevent flood attack
	i := 0
	for {
		i++
		// log.Printf("第 %v 次写入请求", i)
		// log.Printf("发送通道剩余数据 %v 个", len(cache.SendDocker))

		if self.clientSend() {
			read_len, err := conn.Read(request)
			if err != nil || read_len == 0 {
				continue
			}

			data, err := unmarshal(request[:read_len])
			if err != nil {
				continue
			}

			self.clientReceive(data)

			request = make([]byte, 4096) // clear last read content
		}
	}
}

// 发送数据
func (self *Network) clientSend() (gotoRead bool) {
	data := <-cache.SendDocker
	// log.Println("取出数据", data)

	switch data.Type {
	case status.REQTASK:
		gotoRead = true
	case status.LOG:
		gotoRead = false
	}

	self.AutoSend(data)

	return
}

// 处理接收数据
func (self *Network) clientReceive(data *cache.NetData) {
	// log.Println("接收到", *data)
	go func() {
		switch data.Type {
		case status.TASK:
			cache.ReceiveDocker <- data
		case status.LOG:
			self.log(data)
		default:
			self.log(data)
		}
	}()
}

//****************************************通用*******************************************\\

//实时发送点对点信息
func (self *Network) AutoSend(data *cache.NetData) {
	if data.To == "" {
		self.randomSend(data)
	} else {
		self.send(self.Conns[data.To], data)
	}
}

// 随机点对点发信息
func (self *Network) randomSend(data *cache.NetData) {
	self.WaitConn()
	for _, conn := range self.Conns {
		self.send(conn, data)
		return
	}
}

func (self *Network) sendWithClose(conn *Conn, data *cache.NetData) {
	self.send(conn, data)
	conn.Close()
	delete(self.Conns, conn.RemoteAddr())
}

func (self *Network) send(conn *Conn, data *cache.NetData) {
	data.From = self.LocalAddr
	d, err := marshal(data)
	if err != nil {
		log.Println("编码出错了", err)
		return
	}
	conn.Write(d)
	// log.Println("信息已发送", data)
}

// 轮询等待，直到有连接生成
func (self *Network) WaitConn() {
	for len(self.Conns) == 0 {
		time.Sleep(5e8)
	}
}

func (self *Network) GetRunMode() int {
	return self.RunMode
}

func (self *Network) GetConn(key string) *Conn {
	return self.Conns[key]
}

func (self *Network) perHandle(conn net.Conn) *Conn {
	if self.LocalAddr == "" {
		// self.LocalAddr = strings.Split(conn.LocalAddr().String(), ":")[0]
		self.LocalAddr = conn.LocalAddr().String()
	}

	// if _, ok := self.Conns[conn.RemoteAddr().String()]; ok {
	// 	return
	// }
	c := NewConn(conn)
	self.Conns[c.RemoteAddr()] = c
	return c
}

// 打印报告
func (self *Network) log(data *cache.NetData) {
	log.Println(` ********************************************************************************************************************************************** `)
	log.Printf(" * ")
	log.Printf(" *     客户端 [ %s ]    %v", data.From, data.Body)
	log.Printf(" * ")
	log.Println(` ********************************************************************************************************************************************** `)
}
