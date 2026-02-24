package artifact

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
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
	filePath, err := svc.downloadFile(ctx, job, WorkerID)
	log.Println("filepath recieved: ", filePath)
	if err != nil {
		log.Println("error in video worker (downloading file):", err)
		err = svc.setJobStatusFailed(ctx, job)
		// svc.NewJobEventBroker.Publish("job.completed", job)
		return
	}
	vm, err := svc.ffprobe(ctx, filePath)

	if err != nil {
		log.Println("error in video worker: (ffprobe)", err)
		os.Remove(filePath)
		err = svc.setJobStatusFailed(ctx, job)
		// svc.NewJobEventBroker.Publish("job.completed", job)
		return
	}
	filename := filepath.Base(filePath)
	baseName := strings.TrimSuffix(filename, filepath.Ext(filename))
	outputDir := fmt.Sprintf("videos/%v", baseName)
	err = svc.ffmpeg(ctx, filePath, outputDir, vm.Duration, vm.Height, func(percent float64) {
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
		// svc.NewJobEventBroker.Publish("job.completed", job)
		return
	}

	key := uuid.New().String()

	log.Printf("Worker %v uploading HLS to storage...\n", WorkerID)

	// err = svc.StorageSvc.PutHLS(ctx, outputDir, key)

	if err != nil {
		log.Println("error in video worker: (put HLS)", err)
		err = svc.setJobStatusFailed(ctx, job)
		// svc.NewJobEventBroker.Publish("job.completed", job)
		os.RemoveAll(outputDir)
		return
	}
	artifactSize, err := calculateFolderSize(outputDir)
	if err != nil {
		log.Println("error in video worker: (artifact size Calc)", err)
	}
	os.Remove(filePath)
	os.RemoveAll(outputDir)

	videoArtifact := &VideoArtifact{
		ID:             uuid.New(),
		NodeID:         &job.NodeID,
		Key:            &key,
		SizeBytes:      &artifactSize,
		LastAccessedAt: time.Now(),
		Metadata:       vm,
	}

	err = svc.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(videoArtifact).Error; err != nil {
			return err
		}
		if err := tx.Delete(&VideoProcessingJob{NodeID: job.NodeID}).Error; err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		log.Println("error in video worker(video artifact creation): ", err)
	}
	// svc.NewJobEventBroker.Publish("job.completed", job)

}
