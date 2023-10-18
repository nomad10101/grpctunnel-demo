package main

import (
	"bufio"
	"context"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/jhump/grpctunnel"
	"github.com/jhump/grpctunnel/tunnelpb"
	grpc_net_conn "github.com/mitchellh/go-grpc-net-conn"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	grpctunnel_demo "grpctunnel-test/gen_pb"
	"grpctunnel-test/utils"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"sync/atomic"
)

type Endpoint struct {
	grpctunnel_demo.UnimplementedEndpointServiceServer
}

var request_count = int32(0)

func (this *Endpoint) DataPipe(
	stream grpctunnel_demo.EndpointService_DataPipeServer,
) error {

	//ctx, cancel := context.WithCancel(stream.Context())
	ctx, _ := context.WithCancel(stream.Context())

	//go this.handleIncomingStream(stream, cancel)

	fieldIncomingFunc := func(msg proto.Message) *[]byte {
		//return &msg.(*protocol.Bytes).Data
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

	go handleCommPipe(local, remote)

	for {
		// exit if context is done
		// or continue
		select {
		case <-ctx.Done():
			//log.Println("Stream is cancelled!")
			log.Println("DataPipe stream exited")
			return ctx.Err()
		default:
			break
		}
	}

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

func (this *Endpoint) DataPipe1(
	stream grpctunnel_demo.EndpointService_DataPipeServer,
) error {
	current_count := atomic.AddInt32(&request_count, 1)
	filename := "dst_data_at_" + strconv.Itoa(int(current_count)) + ".bin"

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

	grpctunnel_demo.RegisterEndpointServiceServer(channelServer, &Endpoint{})

	// Open the reverse hub_and_spoke and serve requests.
	if _, err := channelServer.Serve(context.Background()); err != nil {
		log.Fatal(err)
	}
}
