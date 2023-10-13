package main

import (
	"bufio"
	"context"
	"github.com/jhump/grpctunnel"
	"github.com/jhump/grpctunnel/tunnelpb"
	"google.golang.org/grpc"
	grpctunnel_demo "grpctunnel-test/gen_pb"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	syscall "syscall"
)

type Hub struct {
	client grpctunnel_demo.EndpointServiceClient
}

func (this *Hub) uplaodFile(filename string) {
	log.Println("Begin Tx: ", filename)

	fi, err := os.Open(filename)
	if err != nil {
		panic(err)
	}

	defer fi.Close()

	stream, err := this.client.DataPipe(context.Background())

	// make a read buffer
	reader := bufio.NewReader(fi)

	buf := make([]byte, 16384)

	for {
		// read a chunk
		n, err := reader.Read(buf)
		if err != nil && err != io.EOF {
			panic(err)
		}

		stream.Send(&grpctunnel_demo.HubMessage{
			DataChunk: buf[:n],
		})

		if n == 0 {
			break
		}
	}

	stream.CloseSend()

	log.Println("Done Tx: ", filename)
}

func main() {
	log.Println("Starting network server")

	handler := grpctunnel.NewTunnelServiceHandler(
		grpctunnel.TunnelServiceHandlerOptions{},
	)

	svr := grpc.NewServer()
	tunnelpb.RegisterTunnelServiceServer(svr, handler.Service())

	hub := &Hub{
		client: grpctunnel_demo.NewEndpointServiceClient(handler.AsChannel()),
	}

	// Just for testing. To init file transfer send SIGHUP to this process.
	//  On macos: pkill -SIGHUP <process-name>
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGHUP)

	go func() {
		for {
			s := <-c
			switch s {
			case syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT:
				return
			case syscall.SIGHUP:
				go hub.uplaodFile("./src_data.bin")
			default:
				return
			}
		}
	}()

	// Start the gRPC server.
	listener, err := net.Listen("tcp", "0.0.0.0:7899")
	if err != nil {
		log.Fatal(err)
	}
	if err := svr.Serve(listener); err != nil {
		log.Fatal(err)
	}

}
