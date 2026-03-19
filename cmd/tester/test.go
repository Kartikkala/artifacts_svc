package main

import (
	"encoding/json"
	"log"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
)

type VideoProcessingJob struct {
	NodeID uuid.UUID `json:"node_id"`
	URL    string    `json:"url"`
}

func main() {
	nc, _ := nats.Connect(nats.DefaultURL)
	defer nc.Close()

	dummyJob := VideoProcessingJob{
		NodeID: uuid.New(),
		URL:    "http://commondatastorage.googleapis.com/gtv-videos-bucket/sample/BigBuckBunny.mp4",
	}
	jobBytes, _ := json.Marshal(dummyJob)

	log.Println("Firing dummy job into queue...")
	// Make sure this string matches your worker's queue subject!
	nc.Publish("video", jobBytes)
}
