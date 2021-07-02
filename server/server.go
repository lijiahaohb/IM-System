package main

import (
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

type Server struct {
	Ip   string
	Port int

	// 当前在线用户列表
	OnlineMap map[string]*User
	// 读写锁 用来保护server中的onlineMap
	mapLock sync.RWMutex

	// 消息广播的channel
	Message chan string
}

// 新建一个server的函数
func NewServer(ip string, port int) *Server {
	return &Server{
		Ip:        ip,
		Port:      port,
		OnlineMap: make(map[string]*User),
		Message:   make(chan string),
	}
}

// 监听Message channel广播消息的方法
func (s *Server) ListenMessager() {
	for {
		msg := <-s.Message

		// 将msg发送给全部的user
		s.mapLock.Lock()
		for _, cli := range s.OnlineMap {
			cli.C <- msg
		}
		s.mapLock.Unlock()
	}
}

func (s *Server) Broadcast(user *User, msg string) {
	sendMsg := "[" + user.Addr + "]" + user.Name + ":" + msg
	s.Message <- sendMsg
}

func (s *Server) Handler(conn net.Conn) {
	// 处理当前连接的业务
	// 创建用户
	user := NewUser(conn, s)

	// 用户上线，将自己添加到server的OnlineMap中
	user.Online()

	// 监听用户是否活跃的channel
	isAlive := make(chan bool)

	// 接收客户端发送的消息
	go func() {
		buf := make([]byte, 4096)
		for {
			num, err := conn.Read(buf)
			if num == 0 {
				user.Offline()
				return
			}
			if err != nil && err != io.EOF {
				fmt.Println("conn read error", err)
			}

			// 提取用户的消息(去除"\n")
			msg := string(buf[:num-1])

			// user处理接收到的消息
			user.DoMessage(msg)
			// 用户的任意消息到来，代表客户活跃
			isAlive <- true
		}
	}()

	// 当前handler阻塞，延长user的生命期
	for {
		select {
		case <-isAlive:
			// 当前用户是活跃的，重置定时器
			// 不做任何事情，只是为了激活select，更新下面的定时器
		case <-time.After(time.Second * 60):
			// 已经超时，将当前的User强制关闭
			user.SendMsg("您已经被踢出了")

			// 销毁资源
			close(user.C)

			// 关闭连接
			conn.Close()

			// 退出当前的handler
			return
		}
	}
}

// 启动服务器的方法
func (s *Server) Start() {
	// socket listen
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%d", s.Ip, s.Port))
	if err != nil {
		fmt.Println("net.Listen() error", err)
		return
	}
	// 延迟关闭listener
	defer listener.Close()

	// 启动监听Message channel的goroutine
	go s.ListenMessager()

	for {
		// accept
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("listener accept error", err)
			continue
		}

		// do handler
		go s.Handler(conn)
	}
}
