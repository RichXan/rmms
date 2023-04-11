package tcp

import (
	"fmt"
	"net"
)

type TcpClient struct {
	conn net.Conn
}

// 创建一个tcp客户端
func NewTcpClient() *TcpClient {
	return &TcpClient{}
}

// 连接服务器
func (tc *TcpClient) Connect(ip string, port int) error {
	var err error
	tc.conn, err = net.Dial("tcp", fmt.Sprintf("%s:%d", ip, port))
	if err != nil {
		return err
	}
	return nil
}

// 发送命令，返回结果
func (tc *TcpClient) Send(cmd []byte) ([]byte, error) {
	//向连接端发送一个数据
	_, err := tc.conn.Write(cmd)
	if err != nil {
		return nil, err
	}
	buf := make([]byte, 1024)
	n, err := tc.conn.Read(buf)
	if err != nil {
		return nil, err
	}
	return buf[:n], nil
}

// 关闭连接
func (tc *TcpClient) Close() error {
	return tc.conn.Close()
}
