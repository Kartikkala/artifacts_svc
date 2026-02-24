package main

import (
	"encoding/json"
	"log"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
)

// We just recreate the struct here to serialize it easily
type VideoJob struct {
	NodeID string `json:"node_id"`
	URL    string `json:"url"`
}

func main() {
	// 1. Connect to your local NATS server
	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		log.Fatal(err)
	}
	defer nc.Close()

	// 2. Create a perfectly valid dummy payload
	// Using a reliable public test MP4 (15MB)
	job := VideoJob{
		NodeID: uuid.New().String(),
		URL:    "http://commondatastorage.googleapis.com/gtv-videos-bucket/sample/ForBiggerBlazes.mp4",
	}

	payload, _ := json.Marshal(job)

	// 3. Fire the payload into the void!
	err = nc.Publish("video", payload)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Successfully published test job!\nUUID: %s\n", job.NodeID)
}
