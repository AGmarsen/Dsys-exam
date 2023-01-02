package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"

	gRPC "github.com/AGmarsen/Dsys-exam/proto"

	"google.golang.org/grpc"
)

// Same principle as in client. Flags allows for user specific arguments/values

var server gRPC.SClient         //the server
var ServerConn *grpc.ClientConn //the server connection

func main() {
	var port = 5000
	var conn *grpc.ClientConn
	log.Printf("Trying to dial: %v\n", port)
	conn, err := grpc.Dial(fmt.Sprintf(":%v", port), grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Fatalf("Could not connect: %s", err)
	} else {
		log.Println("Connection established")
	}
	defer conn.Close()
	server = gRPC.NewSClient(conn)

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		send(scanner.Text())
	}
}

func send(message string) { //here we send our messages to the server
	res, err := server.Bid(context.Background(), &gRPC.M{Name: message, Amount: -1})
	if err != nil {
		log.Printf("%v", err)
	} else {
		log.Println(res.Name)
		log.Printf("%d", res.Amount)
	}
}
