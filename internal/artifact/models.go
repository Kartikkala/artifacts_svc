package artifact

import (
	"time"

	"github.com/google/uuid"
)

type ProcessingStatus string

const (
	StatusPending    ProcessingStatus = "pending"
	StatusProcessing ProcessingStatus = "processing"
	StatusReady      ProcessingStatus = "ready"
	StatusFailed     ProcessingStatus = "failed"
)

type VideoProcessingJob struct {
	NodeID     uuid.UUID        `gorm:"primaryKey"`
	URL        string           `gorm:"url"`
	Status     ProcessingStatus `gorm:"column:status"`
	CreatedAt  time.Time        `gorm:"column:created_at"`
	AccessedAt time.Time        `gorm:"column:accessed_at"`
}
