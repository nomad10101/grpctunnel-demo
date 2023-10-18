package main

import (
	"bufio"
	"context"
	"errors"
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

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	stream, err := this.client.DataPipe(ctx)
	if err != nil {
		log.Println("Fail Tx: ", err)
		return
	}

	// make a read buffer
	reader := bufio.NewReader(fi)

	buf := make([]byte, 16384)

	for {
		// read a chunk
		n, err := reader.Read(buf)
		if err != nil && err != io.EOF {
			panic(err)
		}

		err = stream.Send(&grpctunnel_demo.HubMessage{
			DataChunk: buf[:n],
		})
		if err != nil {
			log.Println("Fail Tx: ", err)
			return
		}

		if n == 0 {
			break
		}
	}

	err = stream.CloseSend()
	if err != nil {
		log.Println("Fail Tx: ", err)
		return
	}

	// Now we must exhaust the stream for RPC to complete
	// TODO: if it is possible for server to send messages *while*
	//       we are uploading above, then must be done in another goroutine
	//       to avoid any chances of deadlock -- like where the server is
	//       trying to send a message and waiting on the client to receive
	//       it before it accepts another upload chunk.
	for {
		_, err = stream.Recv()
		if errors.Is(err, io.EOF) {
			break // done
		}
		if err != nil {
			if err != nil {
				log.Println("Fail Tx: ", err)
				return
			}
		}
	}

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
