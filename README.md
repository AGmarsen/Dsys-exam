# Dsys-exam

## Running the program

Open two terminals for the two nodes and a terminal for each client.

In the server terminals use the command: go run \[path to server.go\] \[number\]

Replace \[number\] with 0 for the leader node and 1 for the backup node.

In the client terminals use the command: go run \[path to client.go\]

Once connection is established you can now enter text lines in the client terminals.

You can simulate a node crash by using ctrl+C in a node terminal or by entering the line "end".
