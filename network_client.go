package main

import (
	"context"
	"github.com/golang/protobuf/proto"
	"github.com/jhump/grpctunnel"
	"github.com/jhump/grpctunnel/tunnelpb"
	grpc_net_conn "github.com/mitchellh/go-grpc-net-conn"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	grpctunnel_demo "grpctunnel-test/gen_pb"
	"grpctunnel-test/utils"
	"log"
	"time"
)

type Endpoint struct {
	grpctunnel_demo.UnimplementedEndpointServiceServer
}

var request_count = int32(0)

func (this *Endpoint) DataPipe(
	stream grpctunnel_demo.EndpointService_DataPipeServer,
) error {
	fin := make(chan bool)

	//ctx, cancel := context.WithCancel(stream.Context())
	ctx, _ := context.WithCancel(stream.Context())

	fieldIncomingFunc := func(msg proto.Message) *[]byte {
		return &msg.(*grpctunnel_demo.HubMessage).DataChunk
	}

	fieldOutgoingFunc := func(msg proto.Message) *[]byte {
		//return &msg.(*protocol.Bytes).Data
		return &msg.(*grpctunnel_demo.EndpointMessage).DataChunk
	}

	// Wrap our conn around the response.
	local := &grpc_net_conn.Conn{
		Stream:   stream,
		Request:  &grpctunnel_demo.EndpointMessage{},
		Response: &grpctunnel_demo.HubMessage{},
		Encode:   grpc_net_conn.SimpleEncoder(fieldOutgoingFunc),
		Decode:   grpc_net_conn.SimpleDecoder(fieldIncomingFunc),
	}

	remote := utils.TcpConnect("localhost", "22")

	go utils.HandleCommPipe(local, remote)

	for {
		// exit if context is done
		// or continue
		select {
		case <-ctx.Done():
			//log.Println("Stream is cancelled!")
			log.Println("DataPipe stream exited")
			return ctx.Err()
		case <-fin:
			break
		default:
			time.Sleep(1 * time.Millisecond)
			break
		}
	}
}

func main() {
	log.Println("Starting network client")
	// Dial the server.
	cc, err := grpc.Dial(
		"127.0.0.1:7899",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatal(err)
	}

	// Register services for reverse tunnels.
	tunnelStub := tunnelpb.NewTunnelServiceClient(cc)
	channelServer := grpctunnel.NewReverseTunnelServer(tunnelStub)

	grpctunnel_demo.RegisterEndpointServiceServer(channelServer, &Endpoint{})

	// Open a tunnel and return a channel.
	stream, err := tunnelStub.OpenTunnel(context.Background())

	ch := grpctunnel.NewChannel(stream)
	if err != nil {
		log.Fatal(err)
	}

	forTunnelClient := grpctunnel_demo.NewHubServiceClient(ch)

	go func() {
		time.Sleep(10000)

		forTunnelClient.CreateTcpListenerAndBlock(context.Background(), &grpctunnel_demo.TcpListenRequest{
			Host: "localhost",
			Port: "50071",
		})
	}()

	// Open the reverse hub_and_spoke and serve requests.
	if _, err := channelServer.Serve(context.Background()); err != nil {
		log.Fatal(err)
	}
}
