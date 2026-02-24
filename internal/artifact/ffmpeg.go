package artifact

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"regexp"
	"strconv"
)

func (svc *Service) ffprobe(
	ctx context.Context,
	FilePath string,
) (*VideoMetadata, error) {
	cmd := exec.CommandContext(ctx,
		"ffprobe",
		"-v", "error",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		FilePath)

	out, err := cmd.Output()

	if err != nil {
		return nil, err
	}

	var ffprobeOutput FFprobeOutput
	var videoMetadata VideoMetadata
	videoMetadata.HasAudio = false
	err = json.Unmarshal(out, &ffprobeOutput)

	if err != nil {
		return nil, err
	}

	for _, stream := range ffprobeOutput.Streams {
		if stream.CodecType == "audio" {
			videoMetadata.HasAudio = true
		} else if stream.CodecType == "video" {
			videoMetadata.Bitrate = ffprobeOutput.Format.Bitrate
			videoMetadata.Codec = stream.CodecName
			videoMetadata.Duration = ffprobeOutput.Format.Duration
			videoMetadata.Height = stream.Height
			videoMetadata.Width = stream.Width
		}
	}

	return &videoMetadata, nil
}

func (svc *Service) ffmpeg(
	ctx context.Context,
	filePath,
	outputDir string,
	duration float64,
	maxResHeight int,
	progress func(percent float64),
) error {

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return err
	}

	cmd := exec.CommandContext(
		ctx,
		"ffmpeg",
		svc.BuildffmpegArgs(filePath, outputDir, duration, maxResHeight)...,
	)

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	scanner := bufio.NewScanner(stderr)
	buf := make([]byte, 0, 1024*1024)
	scanner.Buffer(buf, 1024*1024)
	re := regexp.MustCompile(`time=(\d+):(\d+):(\d+\.\d+)`)

	for scanner.Scan() {
		line := scanner.Text()

		match := re.FindStringSubmatch(line)
		if len(match) == 4 {
			h, _ := strconv.ParseFloat(match[1], 64)
			m, _ := strconv.ParseFloat(match[2], 64)
			s, _ := strconv.ParseFloat(match[3], 64)

			current := h*3600 + m*60 + s
			percent := (current / duration) * 100

			progress(percent)
		}
	}

	return cmd.Wait()
}
