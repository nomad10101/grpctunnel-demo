package utils

import (
	log "log"
	"net"
	"os"
)

func TcpConnect(
	host string,
	port string,
) net.Conn {
	tcpServer, err := net.ResolveTCPAddr("tcp", host+":"+port)
	if err != nil {
		println("ResolveTCPAddr failed:", err.Error())
		os.Exit(1)
	}

	conn, err := net.DialTCP("tcp", nil, tcpServer)
	if err != nil {
		println("Dial failed:", err.Error())
		os.Exit(1)
	}

	return conn
}

func TcpListen(
	host string,
	port string,
) (net.Listener, error) {
	listener, err := net.Listen("tcp", host+":"+port)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	return listener, nil
}
