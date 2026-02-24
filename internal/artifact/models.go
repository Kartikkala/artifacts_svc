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

type ArtifactType string

const (
	Video ArtifactType = "video"
	Audio ArtifactType = "audio"
	Image ArtifactType = "image"
)

type VideoProcessingJob struct {
	NodeID     uuid.UUID        `gorm:"primaryKey"`
	URL        string           `gorm:"url"`
	Status     ProcessingStatus `gorm:"column:status"`
	CreatedAt  time.Time        `gorm:"column:created_at"`
	AccessedAt time.Time        `gorm:"column:accessed_at"`
}

type VideoArtifact struct {
	ID             uuid.UUID      `json:"id" gorm:"id"`
	NodeID         *uuid.UUID     `json:"node_id" db:"node_id"`
	Key            *string        `json:"-" db:"object_storage_key"`            // Only for files, No JSON output
	SizeBytes      *uint64        `json:"size_bytes,omitempty" db:"size_bytes"` // Only for files
	LastAccessedAt time.Time      `json:"last_accessed_at" db:"last_accessed_at"`
	Metadata       *VideoMetadata `json:"metadata" gorm:"type:jsonb"`
}
