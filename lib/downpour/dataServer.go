package downpour

import (
	"comp3200/lib"
	"comp3200/lib/messenger"
	"comp3200/lib/network"
	"log"
	"math/rand"
	"net"
	"strconv"
	"strings"
)

// DataServer is a struct that represents a Downpour data server
type DataServer struct {
	miniBatches [][]network.Record
	index       int
}

// Serve n mini-batches using a messenger
func (ds *DataServer) serveMiniBatches(messenger messenger.Messenger, n int) {
	var batches [][]network.Record
	count := 0
	for count < n {
		index := ds.index + count
		if index >= len(ds.miniBatches) {
			index = 0
			rand.Shuffle(len(ds.miniBatches), func(i, j int) { ds.miniBatches[i], ds.miniBatches[j] = ds.miniBatches[j], ds.miniBatches[i] })
		}
		batches = append(batches, ds.miniBatches[index])
		count++
	}

	// Otherwise serve the minibatches
	// fmt.Println("Serving data")
	messenger.SendInterface(batches)

	ds.index += count
}

// LaunchDataServer starts a data server on a specified address
func LaunchDataServer(address string) {
	l, err := net.Listen("tcp4", address)

	if err != nil {
		log.Println("ERR:", err)
	}

	ds := DataServer{}

	// Initially receive all data
	log.Println("Waiting to be assigned data partition...")
	var data network.Data
	conn, err := l.Accept()
	msg := messenger.NewMessenger(conn)
	msg.ReceiveInterface(&data)

	ds.miniBatches = data.GetMiniBatches(lib.MiniBatchSize)
	log.Println("Assigned data partition")

	// Wait for a model replica to connect
	log.Println("Waiting for model replica...")
	conn, _ = l.Accept()
	msg = messenger.NewMessenger(conn)
	for {
		// Wait for a partition request telling us how many minibatches to send
		n := ds.waitForRequest(msg)

		if n > 0 {

			// Serve request
			ds.serveMiniBatches(msg, n)
		}
	}
}

// Wait for a data request to come in before continuing
func (ds *DataServer) waitForRequest(messenger messenger.Messenger) int {
	// fmt.Println("Waiting for data request")
	var msg string
	messenger.ReceiveMessage(&msg)

	parts := strings.Split(msg, " ")

	// If message reads REQ then exit and serve the partition
	if parts[0] == "REQ" {
		count, _ := strconv.ParseInt(parts[1], 10, 32)
		// fmt.Println("Received data request for", count, "batches")
		return int(count)
	}

	return 0
}
