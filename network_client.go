package main

import (
	"bufio"
	"context"
	"github.com/jhump/grpctunnel"
	"github.com/jhump/grpctunnel/tunnelpb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	grpctunnel_demo "grpctunnel-test/gen_pb"
	"io"
	"log"
	"os"
	"strconv"
)

type EndpointService struct {
	grpctunnel_demo.UnimplementedEndpointServiceServer
}

var request_count = 0

func (this *EndpointService) DataPipe(
	stream grpctunnel_demo.EndpointService_DataPipeServer,
) error {
	filename := "dst_data_at_" + strconv.Itoa(request_count) + ".bin"

	request_count++

	log.Println("Begin Rx: ", filename)

	fo, err := os.Create(filename)
	if err != nil {
		panic(err)
	}

	defer fo.Close()

	// make a write buffer
	writer := bufio.NewWriter(fo)

	for {
		msg, err := stream.Recv()

		if err == io.EOF {
			//exitLoop = true
			break
		} else if err != nil {
			panic(err)
		}

		// write a chunk
		if _, err := writer.Write(msg.DataChunk); err != nil {
			panic(err)
		}
	}

	if err = writer.Flush(); err != nil {
		panic(err)
	}

	log.Println("End Rx: ", filename)

	return nil
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

	grpctunnel_demo.RegisterEndpointServiceServer(channelServer, &EndpointService{})

	// Open the reverse hub_and_spoke and serve requests.
	if _, err := channelServer.Serve(context.Background()); err != nil {
		log.Fatal(err)
	}
}
