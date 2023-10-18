package main

import (
	"context"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/jhump/grpctunnel"
	"github.com/jhump/grpctunnel/tunnelpb"
	grpc_net_conn "github.com/mitchellh/go-grpc-net-conn"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
	grpctunnel_demo "grpctunnel-test/gen_pb"
	"grpctunnel-test/utils"
	"io"
	"log"
	"net"
)

type Hub struct {
	client grpctunnel_demo.EndpointServiceClient
}

func (this *Hub) CreateTcpListenerAndBlock(
	context context.Context,
	// request *grpctunnel_demo.TcpListenRequest,
) (*emptypb.Empty, error) {
	log.Println("START LISTEN")

	listener, err := utils.TcpListen("localhost", "50071") //(request.GetHost(), request.GetPort())
	if err == nil {
		// close listener on exit
		defer listener.Close()

		go this.tcpAcceptAndServe(listener)

		exitLoop := false

		for !exitLoop {
			select {
			case <-context.Done():
				log.Println("Tcp Listener exit")
				exitLoop = true
			}
		}
	} else {
		log.Println("Listen failed. Error: ", err)
	}

	log.Println("END LISTEN")

	return &emptypb.Empty{}, nil
}

func (this *Hub) tcpAcceptAndServe(
	listener net.Listener,
) {
	var index = 0
	exitLoop := false

	for !exitLoop {
		conn, err := listener.Accept()

		index++
		log.Println("Accepted: ", index, conn)

		if err == nil {
			stream, _ := this.client.DataPipe(context.Background())

			//_, cancelFunc := context.WithCancel(stream.Context())

			///
			// We need to create a callback so the conn knows how to decode/encode
			// arbitrary byte slices for our proto type.
			fieldOutgoingFunc := func(msg proto.Message) *[]byte {
				//return &msg.(*protocol.Bytes).Data
				return &msg.(*grpctunnel_demo.HubMessage).DataChunk
			}

			fieldIncomingFunc := func(msg proto.Message) *[]byte {
				//return &msg.(*protocol.Bytes).Data
				return &msg.(*grpctunnel_demo.EndpointMessage).DataChunk
			}

			// Wrap our conn around the response.
			remote := &grpc_net_conn.Conn{
				Stream:   stream,
				Request:  &grpctunnel_demo.HubMessage{},
				Response: &grpctunnel_demo.EndpointMessage{},
				Encode:   grpc_net_conn.SimpleEncoder(fieldOutgoingFunc),
				Decode:   grpc_net_conn.SimpleDecoder(fieldIncomingFunc),
			}

			//println(fieldOutgoingFunc)

			///

			/*
				if err == nil {
					stream.Send(&protocol.HubMessage{
						MessagePayload: &protocol.HubMessage_TcpConnectRequest{},
					})

					go handleStream(conn, stream, cancelFunc, index)
				} else {
					log.Println("GRPC stream init failed. Error: ", err)
					exitLoop = true
				}
			*/

			go handleCommPipe(conn, remote)
		} else {
			log.Println("Tcp Accept failed. Error: ", err)
			exitLoop = true
		}
	}

	log.Println("Exiting listen")
}

func handleCommPipe(
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

	go hub.CreateTcpListenerAndBlock(context.Background())

	// Start the gRPC server.
	listener, err := net.Listen("tcp", "0.0.0.0:7899")
	if err != nil {
		log.Fatal(err)
	}
	if err := svr.Serve(listener); err != nil {
		log.Fatal(err)
	}

}
