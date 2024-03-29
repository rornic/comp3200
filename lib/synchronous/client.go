package synchronous

import (
	"comp3200/lib"
	"comp3200/lib/messenger"
	"comp3200/lib/network"
	"log"

	"gonum.org/v1/gonum/mat"
)

type client struct {
	model *network.Network
	data  network.Data
}

// LaunchClient starts a synchronous model replica client and connects to a parameter ser ver
func LaunchClient(paramAddress string) {
	data := network.LoadData()

	minibatches := data.GetMiniBatches(lib.MiniBatchSize)
	idx := 0
	batchesPerUpdate := 7
	epochs := 0

	param := messenger.Connect(paramAddress)

	client := client{}

	param.SendMessage("MDL")
	var networkConfig network.NetworkConfig
	param.ReceiveInterface(&networkConfig)
	client.model = network.NewNetworkFromConfig(networkConfig)
	log.Println("Retrieved model configuration")
	for {
		client.receiveParameters(param)

		// Zero all parameters to get empty matrices to accumulate deltas in
		weights, biases := client.model.ZeroedParameters()

		// Do minibatches
		for i := 0; i < batchesPerUpdate; i++ {
			batch := minibatches[idx]
			w, b := client.model.TrainAndUpdate(batch)

			// Sum deltas over minibatches
			for j := 0; j < len(w); j++ {
				weights[j].Add(&weights[j], &w[j])
				biases[j].AddVec(&biases[j], &b[j])
			}

			idx++
			if idx >= len(minibatches) {
				minibatches = data.GetMiniBatches(lib.MiniBatchSize)
				idx = 0
				epochs++
			}
		}

		// Send update and wait for continue signal
		client.sendDeltas(param, weights, biases)
		client.waitForContinue(param)
	}
}

func (mr *client) receiveParameters(msg messenger.Messenger) {
	// Send request to parameter server
	msg.SendMessage("REQ")

	// Retrieve weights and biases for each layer from parameter server
	var weights []mat.Dense
	var biases []mat.VecDense

	msg.ReceiveInterface(&weights)
	msg.ReceiveInterface(&biases)

	mr.model.SetParameters(weights, biases)
}

func (mr *client) sendDeltas(msg messenger.Messenger, weights []mat.Dense, biases []mat.VecDense) {
	// send weight and bias deltas to parameter server
	msg.SendMessage("UPD")
	msg.SendInterface(weights)
	msg.SendInterface(biases)
}

func (mr *client) waitForContinue(msg messenger.Messenger) {
	cmd := ""
	for cmd != "CON" {
		msg.ReceiveMessage(&cmd)
	}
}
