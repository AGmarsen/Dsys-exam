package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"sync"
	"time"

	gRPC "github.com/AGmarsen/Dsys-exam/proto"

	"google.golang.org/grpc"
)

type Server struct {
	gRPC.UnimplementedDictionaryServer
	ownPort    int
	dictionary map[string]string
	otherPort  int //for the primary this will be the port for the backup vice versa
	otherNode  gRPC.DictionaryClient
	isLeader   bool
	inbox      chan bool
	mutex      sync.Mutex
}

func main() {

	arg1, _ := strconv.ParseInt(os.Args[1], 10, 32)
	peer := 5000 + (arg1+1)%2 //if ownPort 5000, otherPort = 5001 vice versa
	server := &Server{
		ownPort:    int(arg1 + 5000),
		dictionary: make(map[string]string),
		otherPort:  int(peer),
		inbox:      make(chan bool),
	}
	server.isLeader = server.ownPort < server.otherPort

	// Create listener tcp on given port or default port 5400
	list, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", server.ownPort))
	if err != nil {
		log.Printf("Failed to listen on port %d: %v", server.ownPort, err) //If it fails to listen on the port, run launchServer method again with the next value/port in ports array
		return
	}

	grpcServer := grpc.NewServer()

	gRPC.RegisterDictionaryServer(grpcServer, server)

	log.Printf("Server: Listening at %v", list.Addr())

	go func() {
		if err := grpcServer.Serve(list); err != nil {
			log.Fatalf("failed to serve %v", err)
		}
	}()

	var conn *grpc.ClientConn
	log.Printf("Trying to dial: %v\n", server.otherPort)
	conn, er := grpc.Dial(fmt.Sprintf(":%v", server.otherPort), grpc.WithInsecure(), grpc.WithBlock())
	if er != nil {
		log.Fatalf("Could not connect: %s", err)
	}
	defer conn.Close()
	server.otherNode = gRPC.NewDictionaryClient(conn)
	log.Println("Connection established")
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		if scanner.Text() == "end" {
			break
		}
	}
}

func (s *Server) Add(ctx context.Context, entry *gRPC.Entry) (*gRPC.Ack, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	log.Printf("Received: %s: %s", entry.Word, entry.Definition)
	if entry.Word == "" || entry.Definition == "" {
		return &gRPC.Ack{Success: false}, nil
	}
	if !s.isLeader { //this happens in the backup node when leader wants to update it and if the leader loses its backup
		s.dictionary[entry.Word] = entry.Definition
		return &gRPC.Ack{Success: true}, nil
	}
	log.Println("Updating backup...")
	go func() {
		_, err := s.otherNode.Add(context.Background(), entry) //update backup
		if err != nil {
			log.Printf("Backup responded with: %v", err)
			s.inbox <- false //if backup gives an error we assume it to be faulty and we declare it dead
		} else {
			log.Println("Backup updated")
			s.inbox <- true
		}
	}()

	result := s.WaitForResponse() //await response from goroutine above
	
	if !result { //if server gave an error declare it dead
		s.isLeader = false //no longer has to notify backup and act as leader
		log.Println("Backup declared dead.")
	}
	s.dictionary[entry.Word] = entry.Definition
	return &gRPC.Ack{Success: true}, nil
}

func (s *Server) Read(ctx context.Context, key *gRPC.Key) (*gRPC.Value, error) {
	log.Printf("Received: %s", key.Word)
	def := s.dictionary[key.Word]
	if def == "" {
		return &gRPC.Value{Definition: "No definition found"}, nil
	}
	return &gRPC.Value{Definition: def}, nil
}

func (s *Server) WaitForResponse() bool {
	end := time.Now().Add(3000 * time.Millisecond) //give the backup 3 seconds to respond
	kill := make(chan bool)
	for {
		select { //waits for a response in the inbox or a kill order
		case result := <-s.inbox:
			return result
		case <-kill:
			log.Println("Backup node failed to respond. Declaring it dead.")
			s.isLeader = false //no longer has to notify backup and act as leader
			return true
		default: //if too much time passes. Send kill order
			if time.Now().After(end) {
				kill <- true
			}
		}
	}
}
