// #Title           stomp.go
// #Description     stomp 链接，订阅，发布相关函数
// #Author          xiaofeng 2022-05-19 15:26:56
// #Update          xiaofeng 2022-05-19 15:26:59
package websocket

import (
	"errors"
	"fmt"
	"sync"
	"time"

	stomp "github.com/go-stomp/stomp/v3"
)

// IStompWebSocket websocket操作类
type IStompWS interface {
	Pubscribe(topic string, msg []byte) error
	AddSub(topic string, CallbackFunc func([]byte))
}

// stomp websocket 配置
type StompWSConfig struct {
	Host    string `mapstructure:"host" yaml:"host"`
	Port    string `mapstructure:"port" yaml:"port"`
	Name    string `mapstructure:"name" yaml:"name"`
	Passwd  string `mapstructure:"passwd" yaml:"passwd"`
	Timeout int    `mapstructure:"timeout" yaml:"timeout"`
	Debug	bool   `mapstructure:"debug" yaml:"debug"`
}

// 订阅参数结构体
type SubscribeParam struct {
	Topic        string
	CallbackFunc func([]byte)
}

// oss 结构体
type StompWS struct {
	config  *StompWSConfig
	conn    *stomp.Conn
	subList []SubscribeParam
}

// #title       stompWSConn
// #description 创建一个新的 stomp 连接句柄
// #author      xiaofeng    2022-05-17 14:23:19
// #param       config      *StompWSConfig  stomp websocket 配置
// #return
func (ws *StompWS) stompWSConn(config *StompWSConfig) error {
	// 连接
	newconn, err := ws.connect(config.Host, config.Port, config.Name, config.Passwd)
	if err != nil {
		fmt.Println(err.Error())
		return err
	}

	if newconn == nil {
		fmt.Println("newconn is nil")
		return errors.New("newconn is nil")
	}

	ws.conn = newconn
	return nil
}

// #title       connect
// #description 连接
// #author      xiaofeng    2022-05-17 14:34:44
// #param       host       string      stomp服务器地址
// #param       port       string      stomp服务器端口
// #param       name       string      stomp服务器用户名
// #param       passwd     string      stomp服务器密码
// #return      error       错误信息
func (ws *StompWS) connect(host, port, name, passwd string) (*stomp.Conn, error) {
	addr := fmt.Sprintf("%s:%s", host, port)
	fmt.Printf("conn to addr:%s, name:%s, passwd:%s \n", addr, name, passwd)

	var options = []func(*stomp.Conn) error{
		stomp.ConnOpt.Login(name, passwd),
		//设置读写超时，超时时间为1个小时
		stomp.ConnOpt.HeartBeat(7200*time.Second, 7200*time.Second),
		// stomp.ConnOpt.HeartBeat(time.Duration(ws.config.Timeout)*time.Second, time.Duration(ws.config.Timeout)*time.Second),
		stomp.ConnOpt.HeartBeatError(360 * time.Second),
		stomp.ConnOpt.Host("/"),
	}

	conn, err := stomp.Dial("tcp", addr, options...)
	if err != nil {
		return nil, err
	}

	return conn, err
}

// #title       subscribe
// #description 订阅
// #author      xiaofeng    2022-05-17 14:34:44
// #param       topic       string      待订阅的topic
//
//	CallbackFunc func([]byte)
//
// #return      error       错误信息
func (ws *StompWS) subscribe(topic string, CallbackFunc func([]byte), wg *sync.WaitGroup) {
	sub, err := ws.conn.Subscribe(topic, stomp.AckAuto)
	if err != nil {
		fmt.Printf("##### sub %s Error: %v #####", topic, err)
		return
	}

	for {
		// 断线，推出进程
		if !sub.Active() {
			fmt.Printf("##### topic %s Error: connection closed #####", topic)
			wg.Done()
			return
		}

		select {
		case v := <-sub.C:
			go CallbackFunc(v.Body)
		case <-time.After(10 * time.Minute):
		}
	}
}

// #title       Pubscribe
// #description 发布
// #author      xiaofeng    2022-05-17 14:34:44
// #param       topic       string      待订阅的topic
// #param       msg         []byte      发布的消息
// #return      error       错误信息
func (ws *StompWS) Pubscribe(topic string, msg []byte) error {
	// 判断ws连接是否存在
	if ws.conn == nil {
		// 打印错误日志
		fmt.Println("ws conn is nil")
		return errors.New("websocket is not connection")
	}

	// 打印日志
	// fmt.Printf("##### Publish topic: %s, msg: %s ##### \n", topic, string(msg))

	err := ws.conn.Send(topic, "text/plain", msg)
	if err != nil {
		fmt.Println("Pubscribe erorr: " + err.Error())
	}

	return nil
}

// #title       close
// #description 关闭连接
// #author      xiaofeng    2022-05-17 14:34:44
// #return      error       错误信息
func (ws *StompWS) close() error {
	// 打印log
	fmt.Println("stomp Close")

	ws.conn.Disconnect()
	ws.conn = nil
	return nil
}

// #title
// #description	添加订阅
// #author      xiaofeng    2022-05-17 14:34:44
// #param       topic       string      待订阅的topic
// #param       CallbackFunc func([]byte)
// #return      error       错误信息
func (ws *StompWS) AddSub(topic string, CallbackFunc func([]byte)) {
	// 生成SubscribeParam
	subParam := SubscribeParam{
		Topic:        topic,
		CallbackFunc: CallbackFunc,
	}

	// 添加到订阅列表
	ws.subList = append(ws.subList, subParam)
}

// #title       WebsocketStart
// #description websocket初始化
// #author      xiaofeng    2022-05-17 14:34:44
// #param       ws      			*StompWS  			stomp 结构体指针
// #param       websocketConfig 	*WebsocketConfig  	websocket 配置
// #return      None
func (ws *StompWS) WebsocketStart() {
	go func() {
		var wg sync.WaitGroup
		// 重连
		for {
			// 连接stomp
			error := ws.stompWSConn(ws.config)
			if error != nil {
				// 打印log
				fmt.Println("cant not connect to websocket: " + error.Error())

				// 等待1秒
				time.Sleep(time.Second)
				continue
			}

			// 遍历订阅列表
			for _, subParam := range ws.subList {
				// 订阅
				wg.Add(1)
				go ws.subscribe(subParam.Topic, subParam.CallbackFunc, &wg)
			}

			// 等待所有goroutine结束
			wg.Wait()

			// 关闭连接
			ws.close()
		}
	}()
}

// #title       WebsocketStart
// #description websocket初始化
// #author      xiaofeng    2022-05-17 14:34:44
// #param       websocketConfig 	*WebsocketConfig  	websocket 配置
// #return      None
func (ws *StompWS) WebsocketInit(stompWSConfig *StompWSConfig) {
	ws.config = stompWSConfig
	ws.stompWSConn(ws.config)
}

func NewStompWs() *StompWS {
	return &StompWS{}
}
