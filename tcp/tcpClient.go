package tcp

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"sync"
	"time"
)

type IConn interface {
	Close() error
}

// Conn 对应每个连接
type Conn struct {
	addr    string       // 地址
	tcp     *net.TCPConn // tcp连接实例, 可以是其他类型
	ctx     context.Context
	writer  *bufio.Writer
	cnlFun  context.CancelFunc // 用于通知ctx结束
	retChan *sync.Map          // 存放通道结果集合的map, 属于统一连接
	err     error
}

type Option struct {
	addr        string
	size        int
	readTimeout time.Duration
	dialTimeout time.Duration
	keepAlive   time.Duration
}

// TCP 客户端
func main(message string) {
	conn, err := net.Dial("tcp", "127.0.0.1:9999")
	if err != nil {
		fmt.Println("err : ", err)
		return
	}
	defer conn.Close() // 关闭TCP连接
	for {
		_, err := conn.Write([]byte(message)) // 发送数据
		if err != nil {
			return
		}
		buf := [512]byte{}
		n, err := conn.Read(buf[:])
		if err != nil {
			fmt.Println("recv failed, err:", err)
			return
		}
		fmt.Println(string(buf[:n]))
	}
}

// 为Conn实现Close()函数签名 关闭连接, 关闭消息通道
func (c *Conn) Close() (err error) {
	// 执行善后
	if c.cnlFun != nil {
		c.cnlFun()
	}

	// 关闭tcp连接
	if c.tcp != nil {
		err = c.tcp.Close()
	}

	// 关闭消息通道
	if c.retChan != nil {
		c.retChan.Range(func(key, value interface{}) bool {
			// 根据具体业务断言转换通道类型
			if ch, ok := value.(chan string); ok {
				close(ch)
			}
			return true
		})
	}
	return
}

func NewConn(opt *Option) (c *Conn, err error) {
	// 初始化连接
	c = &Conn{
		addr:    opt.addr,
		retChan: new(sync.Map),
		//err: nil,
	}

	defer func() {
		if err != nil {
			if c != nil {
				c.Close()
			}
		}
	}()

	// 拨号
	var conn net.Conn
	if conn, err = net.DialTimeout("tcp", opt.addr, opt.dialTimeout); err != nil {
		return
	} else {
		c.tcp = conn.(*net.TCPConn)
	}

	c.writer = bufio.NewWriter(c.tcp)

	//if err = c.tcp.SetKeepAlive(true); err != nil {
	if err = c.tcp.SetKeepAlive(false); err != nil {
		return
	}
	if err = c.tcp.SetKeepAlivePeriod(opt.keepAlive); err != nil {
		return
	}
	if err = c.tcp.SetLinger(0); err != nil {
		return
	}

	// 创建上下文管理
	c.ctx, c.cnlFun = context.WithCancel(context.Background())

	// 异步接收结果到相应的结果集
	// go receiveResp(c)

	return
}
