package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	gRPC "github.com/AGmarsen/Dsys-exam/proto"

	"google.golang.org/grpc"
)

// Same principle as in client. Flags allows for user specific arguments/values

var server gRPC.DictionaryClient //the server
var conn *grpc.ClientConn        //the server connection
var serverPort = 5000
var inbox chan bool

func main() {
	connectToPort(serverPort)
	defer conn.Close()
	inbox = make(chan bool)
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Println("Type \"<word>: <definition>\" to add an entry to the dictionary.")
	fmt.Println("Type \"<word>:\" to read an entry from the dictionary.")
	for scanner.Scan() {
		formatted, valid := format(scanner.Text())
		if !valid {
			continue
		}
		if formatted[1] == "" {
			read(formatted[0])
		} else {
			add(formatted[0], formatted[1])
		}
	}
}

func connectToPort(port int) {
	if port > 5001 {
		log.Println("Too many failures :(")
		return
	}
	log.Printf("Trying to dial: %v\n", port)
	conn, err := grpc.Dial(fmt.Sprintf(":%v", port), grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Fatalf("Could not connect: %s", err)
	} else {
		log.Println("Connection established")
	}
	server = gRPC.NewDictionaryClient(conn)
}

func add(word string, definition string) { //here we send our messages to the server
	response := make(chan string)
	go func() {
		ack, err := server.Add(context.Background(), &gRPC.Entry{Word: word, Definition: definition})
		if err != nil {
			log.Printf("Server responded with: %v", err)
			inbox <- false
			return
		} else {
			inbox <- true
			if ack.Success {
				response <- "Successfully added."
			} else {
				response <- "Something went wrong."
			}
		}
	}()
	success := WaitForResponse()
	if success {
		log.Println(<-response)
		return
	}
	log.Println("Redirecting to backup and trying again...")
	serverPort++
	connectToPort(serverPort)
	add(word, definition)
}

func read(word string) {
	response := make(chan string)
	go func() {
		ack, err := server.Read(context.Background(), &gRPC.Key{Word: word})
		if err != nil {
			log.Printf("Server responded with: %v", err)
			inbox <- false
			return
		} else {
			inbox <- true
			response <- ack.Definition
		}
	}()
	success := WaitForResponse()
	if success {
		log.Printf("%s: %s", word, <-response)
		return
	}
	log.Println("Redirecting to backup and trying again...")
	serverPort++
	connectToPort(serverPort)
	read(word)
}

func format(raw string) ([]string, bool) {
	var formatted = strings.Split(raw, ":")
	if len(formatted) != 2 {
		fmt.Println("You must use exactly one colon.")
		return append(append(make([]string, 2), ""), ""), false
	}
	formatted[0] = strings.Trim(formatted[0], " ")
	formatted[1] = strings.Trim(formatted[1], " ")
	return formatted, true
}

func WaitForResponse() bool {
	end := time.Now().Add(3000 * time.Millisecond) //give the backup 3 seconds to respond
	kill := make(chan bool)
	for {
		select { //waits for a response in the inbox or a kill order
		case result := <-inbox:
			return result
		case <-kill:
			log.Println("Server failed to respond")
			return false
		default: //if too much time passes. Send kill order
			if time.Now().After(end) {
				kill <- true
			}
		}
	}
}
