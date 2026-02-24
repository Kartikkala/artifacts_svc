package artifact

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

func (svc *Service) downloadFile(
	ctx context.Context,
	job *VideoProcessingJob,
	WorkerID uint8,
) (string, error) {
	// Create a Context-Aware HTTP Request using the Presigned URL
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, job.URL, nil)
	if err != nil {
		log.Printf("[Worker %d] error creating HTTP request: %v\n", WorkerID, err)
		return "", err
	}

	// Execute the Download
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("[Worker %d] error downloading file: %v\n",
			WorkerID,
			err,
		)
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("download failed with HTTP status: %s", resp.Status)
		log.Printf("[Worker %d] %v\n", WorkerID, err)
		return "", err
	}

	if err := os.MkdirAll("videos", 0755); err != nil {
		log.Printf("[Worker %d] error creating videos directory: %v\n", WorkerID, err)
		return "", err
	}

	filepath := fmt.Sprintf("videos/%d_%s.tmp", WorkerID, job.NodeID.String())
	file, err := os.Create(filepath)
	if err != nil {
		log.Printf("[Worker %d] error in os.Create(): %v\n", WorkerID, err)
		return "", err
	}

	_, err = io.Copy(file, resp.Body)
	file.Close()

	if err != nil {
		log.Printf("[Worker %d] error in io.Copy(): %v\n", WorkerID, err)
		os.Remove(filepath) // Guaranteed cleanup on failure
		return "", err
	}

	return filepath, nil
}

func calculateFolderSize(
	dirPath string,
) (uint64, error) {
	var totalSize int64

	err := filepath.WalkDir(dirPath, func(
		path string,
		d fs.DirEntry,
		err error,
	) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			info, err := d.Info()
			if err != nil {
				return err
			}
			totalSize += info.Size()
		}

		return nil
	})

	if err != nil {
		return 0, err
	}

	return uint64(totalSize), nil
}
