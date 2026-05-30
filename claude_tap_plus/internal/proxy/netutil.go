// Package proxy 提供 HTTP 代理功能，包括请求转发、SSE 流式处理与 Trace 记录。
package proxy

import (
	"net"
)

// netTCPAddr 是 net.TCPAddr 的类型别名，用于获取监听端口的端口号。
type netTCPAddr = net.TCPAddr

// ioListener 在指定地址上创建 TCP 监听器。
func ioListener(addr string) (net.Listener, error) {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}
	return ln, nil
}
