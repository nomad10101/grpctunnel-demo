package main

import (
	"context"
	"github.com/golang/protobuf/proto"
	"github.com/jhump/grpctunnel"
	"github.com/jhump/grpctunnel/tunnelpb"
	grpc_net_conn "github.com/mitchellh/go-grpc-net-conn"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
	grpctunnel_demo "grpctunnel-test/gen_pb"
	"grpctunnel-test/utils"
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

			go utils.HandleCommPipe(conn, remote)
		} else {
			log.Println("Tcp Accept failed. Error: ", err)
			exitLoop = true
		}
	}

	log.Println("Exiting listen")
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
