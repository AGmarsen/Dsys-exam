package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"sync"

	gRPC "github.com/AGmarsen/Dsys-exam/proto"

	"google.golang.org/grpc"
)

type Server struct {
	gRPC.UnimplementedSServer
	ownPort int
	mutex   sync.Mutex
}

func main() {

	arg1, _ := strconv.ParseInt(os.Args[1], 10, 32)
	server := &Server{
		ownPort: int(arg1 + 5000),
	}

	// Create listener tcp on given port or default port 5400
	list, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", server.ownPort))
	if err != nil {
		log.Printf("Failed to listen on port %d: %v", server.ownPort, err) //If it fails to listen on the port, run launchServer method again with the next value/port in ports array
		return
	}

	grpcServer := grpc.NewServer()

	gRPC.RegisterSServer(grpcServer, server)

	log.Printf("Server: Listening at %v", list.Addr())

	go func() {
		if err := grpcServer.Serve(list); err != nil {
			log.Fatalf("failed to serve %v", err)
		}
	}()
	for {
		//keep me alive
		if false {
			break
		}
	}
}

func (s *Server) Bid(ctx context.Context, req *gRPC.M) (*gRPC.M, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	log.Println(req.Name)
	return &gRPC.M{Name: req.Name, Amount: 0}, nil
}
