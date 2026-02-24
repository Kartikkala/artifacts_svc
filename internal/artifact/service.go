package artifact

import (
	"log"

	"github.com/nats-io/nats.go"
	"gorm.io/gorm"
)

func NewService(
	DB *gorm.DB,
	NATSClient *nats.Conn,
	MaxNumberOfWorkers uint8,
) *Service {
	encoder := encoderForVendor(DetectGPUVendor())

	log.Println("Selected encoder : ", encoder)
	DB.AutoMigrate(&VideoArtifact{})
	DB.AutoMigrate(&VideoProcessingJob{})
	return &Service{
		DB:              DB,
		MaxWorkers:      MaxNumberOfWorkers,
		WorkerSemaphore: make(chan struct{}, MaxNumberOfWorkers),
		NATS:            NATSClient,
		VideoEncoder:    encoder,
	}
}
