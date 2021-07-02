package main

import (
	"flag"
	"fmt"
	"io"
	"net"
	"os"
)

type Client struct {
	ServerIp   string
	ServerPort int
	Name       string

	conn net.Conn
	flag int // 当前客户端执行任务的模式
}

func NewClient(serverIp string, serverPort int) *Client {
	// 创建客户端对象
	client := &Client{
		ServerIp:   serverIp,
		ServerPort: serverPort,
		flag:       999,
	}

	// 连接服务器
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", client.ServerIp, client.ServerPort))
	if err != nil {
		fmt.Println("net.Dial error: ", err)
		return nil
	}
	client.conn = conn

	// 返回对象
	return client
}

func (c *Client) getMenu() bool {
	var flag int

	fmt.Println("1. 群聊模式")
	fmt.Println("2. 私聊模式")
	fmt.Println("3. 更新用户名")
	fmt.Println("0. 退出聊天")

	fmt.Scanln(&flag)

	if flag >= 0 && flag <= 3 {
		c.flag = flag
		return true
	} else {
		return false
	}
}

func (c *Client) UpdateName() bool {
	fmt.Println("请输入用户名: ")
	fmt.Scanln(&c.Name)

	sendMsg := "rename|" + c.Name + "\n"
	_, err := c.conn.Write([]byte(sendMsg))
	if err != nil {
		fmt.Println("conn write error: ", err)
		return false
	}
	return true
}

// 公聊模式
func (c *Client) PublicChat() {
	var chatMsg string
	fmt.Println(">>>>>> 请输入聊天内容，键入exit退出 <<<<<<")
	fmt.Scanln(&chatMsg)

	for chatMsg != "exit" {
		if len(chatMsg) != 0 {
			sendMsg := chatMsg + "\n"
			_, err := c.conn.Write([]byte(sendMsg))
			if err != nil {
				fmt.Println("conn write error", err)
				break
			}
		}
		chatMsg = ""
		fmt.Scanln(&chatMsg)
	}
}

// 查询在线用户
func (c *Client) SelectUsers() {
	sendMsg := "who\n"
	_, err := c.conn.Write([]byte(sendMsg))
	if err != nil {
		fmt.Println("conn write error: ", err)
		return
	}
}

// 私聊模式
func (c *Client) PrivateChat() {
	var remoteName string
	var chatMsg string
	// 1. 查询在线用户
	c.SelectUsers()

	fmt.Println(">>>>>> 请输入您要聊天的用户名, 键入exit退出 <<<<<<<")
	fmt.Scanln(&remoteName)

	for remoteName != "exit" {
		fmt.Println(">>>>>> 请输入聊天内容，键入exit退出 <<<<<<")
		fmt.Scanln(&chatMsg)

		for chatMsg != "exit" {
			if len(chatMsg) != 0 {
				sendMsg := "to|" + remoteName + "|" + chatMsg + "\n\n"
				_, err := c.conn.Write([]byte(sendMsg))
				if err != nil {
					fmt.Println("conn write error", err)
					break
				}
			}
			chatMsg = ""
			fmt.Scanln(&chatMsg)
		}
		c.SelectUsers()

		fmt.Println(">>>>>> 请输入您要聊天的用户名, 键入exit退出 <<<<<<<")
		fmt.Scanln(&remoteName)
	}
}

// 接收server回应的消息，并将其显示到标准输出
func (c *Client) DealResponse() {
	io.Copy(os.Stdout, c.conn)
}

func (c *Client) Run() {
	for c.flag != 0 {
		for !c.getMenu() {
		}
		// 根据不同的模式处理不同业务
		switch c.flag {
		case 1:
			// 群聊模式
			c.PublicChat()
		case 2:
			// 私聊模式
			c.PrivateChat()
		case 3:
			// 跟新用户名
			c.UpdateName()
		}

	}
}

var serverIp string
var serverPort int

func init() {
	// 使用方法 ./client -ip 127.0.0.1 -port 8888
	flag.StringVar(&serverIp, "ip", "127.0.0.1", "设置服务器IP地址(默认为127.0.0.1)")
	flag.IntVar(&serverPort, "port", 8888, "设置服务器的端口号(默认为8888)")
}

func main() {
	// 命令行解析
	flag.Parse()

	client := NewClient(serverIp, serverPort)
	if client == nil {
		fmt.Println(">>>>>> 连接服务器失败 <<<<<<")
		return
	}

	// 单独开启goroutine接收显示server返回的消息
	go client.DealResponse()

	fmt.Println(">>>>>> 连接服务器成功 <<<<<<")

	// 启动客户端业务
	client.Run()
}
