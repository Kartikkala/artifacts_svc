package artifact

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"gorm.io/gorm"
)

func (svc *Service) StartWorkers(
	ctx context.Context,
) {
	go svc.jobProducer(ctx)
	var i uint8 = 0
	// Start n workers
	for i = 0; i < svc.MaxWorkers; i++ {
		go svc.runVideoWorker(ctx, i)
	}

	// Throw in n tokens in the channel so that
	// the workers start immediately to check
	// of DB got any pending jobs already
	for i = 0; i < svc.MaxWorkers; i++ {
		select {
		case svc.WorkerSemaphore <- struct{}{}:
		default:
		}
	}
}

func (svc *Service) jobProducer(
	ctx context.Context,
) {
	svc.NATS.Subscribe("video", func(msg *nats.Msg) {
		log.Printf("recieved URL: %s\n", string(msg.Data))
		var job VideoJob
		err := json.Unmarshal(msg.Data, &job)

		if err != nil {
			log.Printf("error in unmarshal %v", err)
			return
		}
		// Fire a new gorutine to return NATS' goroutine
		// so that the service isn't flagged as a slow
		// consumer
		go func(j VideoJob) {
			svc.registerVideoJob(ctx, &j)

			select {
			case svc.WorkerSemaphore <- struct{}{}:
			default:
				log.Println("producer says: bye bye bye!")
			}
		}(job)
	})
}

func (svc *Service) registerVideoJob(
	ctx context.Context,
	job *VideoJob,
) {
	var videoProcessingJob VideoProcessingJob
	nodeID, err := uuid.Parse(job.NodeID)

	if err != nil {
		log.Printf("error in uuid parse(): %s\n", err)
		return
	}
	// Create a new video processing job
	// in Database first
	videoProcessingJob = VideoProcessingJob{
		NodeID:     nodeID,
		URL:        job.URL,
		Status:     StatusPending,
		CreatedAt:  time.Now(),
		AccessedAt: time.Now(),
	}

	err = svc.DB.WithContext(ctx).
		Create(&videoProcessingJob).Error

	if err != nil {
		log.Println("error in job creation:", err)
		return
	}
}

func (svc *Service) runVideoWorker(
	ctx context.Context,
	WorkerID uint8,
) {
	for {
		<-svc.WorkerSemaphore
		for {
			var videoProcessingJob VideoProcessingJob
			// Fetch the Least Recently Accessed video processing
			// job from the database and mark it to processing
			err := svc.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
				if err := tx.Raw(`
        		SELECT * FROM video_processing_jobs
        		WHERE status = ?
        		ORDER BY accessed_at ASC, created_at ASC
        		FOR UPDATE SKIP LOCKED
        		LIMIT 1`, StatusPending).
					Scan(&videoProcessingJob).Error; err != nil {
					return err
				}

				if videoProcessingJob.NodeID == uuid.Nil {
					return gorm.ErrRecordNotFound
				}

				return tx.Model(&VideoProcessingJob{}).
					Where("node_id = ?", videoProcessingJob.NodeID).
					Updates(map[string]any{
						"status":      StatusProcessing,
						"accessed_at": time.Now(),
					}).Error
			})

			if errors.Is(err, gorm.ErrRecordNotFound) {
				log.Println("all DB jobs are done, going back to sleep zzzzzzz....")
				break
			} else if err != nil {
				log.Println("error in job fetch: ", err)
				continue
			}

			// Send that new job to video worker
			svc.videoWorker(ctx, &videoProcessingJob, WorkerID)
		}
	}
}

func (svc *Service) setJobStatusFailed(
	ctx context.Context,
	job *VideoProcessingJob,
) error {
	if err := svc.DB.WithContext(ctx).
		Model(&VideoProcessingJob{}).
		Where("node_id = ?", job.NodeID).
		Update("status", StatusFailed).Error; err != nil {
		return err
	}
	return nil
}

func (svc *Service) videoWorker(
	ctx context.Context,
	job *VideoProcessingJob,
	WorkerID uint8,
) {
	vm, err := svc.ffprobe(ctx, job.URL)
	if err != nil {
		log.Println("error in video worker: (ffprobe)", err)
		err = svc.setJobStatusFailed(ctx, job)
		if err != nil {
			log.Println("error in video worker (DB to set Job fail status): ", err)
		}
		err = svc.NATS.Publish("artifact.fail", []byte(job.NodeID.String()))
		if err != nil {
			log.Println("error in video worker (NATS): ", err)
		}
		return
	}

	upstreamURL := fmt.Sprintf("http://127.0.0.1:9009/hls/%s", job.NodeID.String())
	err = svc.ffmpeg(ctx, job.URL, upstreamURL, vm.Duration, vm.Height, func(percent float64) {
		// TODO send this progress to the frontend!
		log.Printf("Current progress of worker %v: %v\n", WorkerID, percent)
	})

	// TODO 1. Add option to cancel the video
	// conversion and revert the changes

	// TODO 2. Give progress to frontend
	// Add web socket to push progress to frontend

	if err != nil {
		log.Println("error in video worker: (ffmpeg)", err)
		err = svc.setJobStatusFailed(ctx, job)

		if err != nil {
			log.Println("error in video worker (DB to set Job fail status): ", err)
		}

		err = svc.NATS.Publish("artifact.fail", []byte(job.NodeID.String()))
		if err != nil {
			log.Println("error in video worker (NATS): ", err)
		}
		return
	}

	metadataString, err := json.Marshal(vm)
	if err != nil {
		log.Printf("error in marshalling video metadata: %s\n", err.Error())
		err = svc.NATS.Publish("artifact.fail", []byte(job.NodeID.String()))
		if err != nil {
			log.Println("error in video worker (NATS): ", err)
		}
		return
	}

	payload, err := json.Marshal(map[string][]byte{
		"node_id":  []byte(job.NodeID.String()),
		"metadata": metadataString,
	})

	err = svc.NATS.Publish("artifact.success", payload)
	if err != nil {
		log.Println("error in video worker (NATS): ", err)
		return
	}

	if err := svc.DB.WithContext(ctx).Delete(&VideoProcessingJob{NodeID: job.NodeID}).Error; err != nil {
		log.Printf("error in video job deletion: %s\n", err.Error())
	}

}
