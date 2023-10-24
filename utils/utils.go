package utils

import (
	"fmt"
	"io"
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

func HandleCommPipe(
	local net.Conn,
	remote net.Conn,
) {
	defer local.Close()
	chDone := make(chan bool)

	// TODO: Figure out the frequency - size or time
	//localProgressConn := local   //NewProgressConn(local, chRxCount)
	//remoteProgressConn := remote //NewProgressConn(remote, chTxCount)

	// Start remote -> local data transfer
	go func() {
		remoteToLocalTx, err := io.Copy(local, remote)
		if err != nil {
			log.Println(fmt.Sprintf("error while copy remote->local: %s", err))
		}

		fmt.Printf("Data remote -> local: %d \n", int64(remoteToLocalTx/(1024)))

		chDone <- true
	}()

	// Start local -> remote data transfer
	go func() {
		localToRemoteTx, err := io.Copy(remote, local)
		if err != nil {
			log.Println(fmt.Sprintf("error while copy local->remote: %s", err))
		}

		fmt.Printf("Data local -> remote: %d \n", int64(localToRemoteTx/(1024)))

		//fmt.Printf("-> %d\n", localProgressConn.count/(1024*1024))

		chDone <- true
	}()

	<-chDone
}
