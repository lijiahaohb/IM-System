package main

import (
	"net"
	"strings"
)

type User struct {
	Name string
	Addr string
	C    chan string
	conn net.Conn

	// user所属的Server，用来获取server的操作
	server *Server
}

// 创建一个用户的函数
func NewUser(conn net.Conn, server *Server) *User {
	userAddr := conn.RemoteAddr().String()

	// 创建一个user对象
	user := &User{
		Name:   userAddr,
		Addr:   userAddr,
		C:      make(chan string),
		conn:   conn,
		server: server,
	}

	// 启动user对象的监听channel的方法
	go user.ListenMessage()

	return user
}

// 用户上线功能
func (u *User) Online() {
	u.server.mapLock.Lock()
	u.server.OnlineMap[u.Name] = u
	u.server.mapLock.Unlock()

	u.server.Broadcast(u, "已上线")
}

// 用户下线功能
func (u *User) Offline() {
	u.server.mapLock.Lock()
	delete(u.server.OnlineMap, u.Name)
	u.server.mapLock.Unlock()

	u.server.Broadcast(u, "已下线")
}

// 给当前User对应的客户端发送消息
func (u *User) SendMsg(msg string) {
	u.conn.Write([]byte(msg))
}

// 用户处理消息的功能
func (u *User) DoMessage(msg string) {
	if msg == "who" {
		// 查询当前在线用户都有哪些
		u.server.mapLock.Lock()
		for _, user := range u.server.OnlineMap {
			u.SendMsg(user.Name + "在线...\n")
		}
		u.server.mapLock.Unlock()
	} else if len(msg) > 7 && msg[:7] == "rename|" {
		newName := strings.Split(msg, "|")[1]

		// 判断name是否存在
		_, ok := u.server.OnlineMap[newName]
		if ok {
			u.SendMsg("当前用户名已被使用\n")
		} else {
			// 更新server中的OnlineMap中保存的用户名
			u.server.mapLock.Lock()
			delete(u.server.OnlineMap, u.Name)
			u.server.OnlineMap[newName] = u
			u.server.mapLock.Unlock()

			// 更新自己的用户名
			u.Name = newName
			u.SendMsg("您的用户名已经更新为" + u.Name + "\n")
		}
	} else if len(msg) > 4 && msg[:3] == "to|" {
		// 消息格式: to|张三|消息内容
		// 1. 获取对方的用户名
		remoteName := strings.Split(msg, "|")[1]
		if remoteName == "" {
			u.SendMsg("消息格式不正确，消息格式为: 'to|张三|消息内容'\n")
			return
		}

		// 2. 根据用户名获取对方的User对象
		user, ok := u.server.OnlineMap[remoteName]
		if !ok {
			u.SendMsg("用户不在线\n")
			return	
		}

		// 3. 获取消息内容，通过对方的User对象发送消息
		msg := strings.Split(msg, "|")[2]
		if msg == "" {
			u.SendMsg("消息内容为空，请重发\n")
			return
		}
		user.SendMsg(u.Name + "对您说: " + msg)
	} else {
		u.server.Broadcast(u, msg)
	}
}

// 监听当前User channel的方法，一旦有消息就直接将消息发送给客户端
func (u *User) ListenMessage() {
	for {
		msg := <-u.C

		u.conn.Write([]byte(msg + "\n"))
	}
}
