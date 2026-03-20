package main

import (
	"context"
	"log"

	"github.com/nats-io/nats.go"
	"github.com/sirkartik/artifacts_svc/internal/artifact"
	"github.com/sirkartik/artifacts_svc/internal/config"
	"github.com/sirkartik/artifacts_svc/internal/utils"
)

func main() {
	log.Println("Liquid NATS is here...")
	nc, err := nats.Connect(nats.DefaultURL)

	if err != nil {
		log.Println("Error in NATS server connection...", err)
		return
	}
	app, err := config.NewApp()

	if err != nil {
		log.Println("error in initializing app", err)
	}

	artifact_svc := artifact.NewService(app.DB, nc, uint8(4))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	URL, err := utils.ConstructURL(
		app.Cfg.App.ArtifactUpstreamProtocol,
		app.Cfg.App.ArtifactUpstreamAddress,
		app.Cfg.App.ArtifactUpstreamPort,
		app.Cfg.App.ArtifactUpstreamEndpoint,
	)
	if err != nil {
		log.Fatalf("error in URL construct: %s", err.Error())
	}

	artifact_svc.StartWorkers(ctx, URL, app.Cfg.App.WorkerInactivityKillDurationSecs)

	// Block till CTRL+C
	<-ctx.Done()
}
