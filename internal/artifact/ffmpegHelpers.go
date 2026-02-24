package artifact

import (
	"fmt"
	"path/filepath"
	"strings"
)

func (svc *Service) buildFilterComplex(levels []int) string {
	fc := fmt.Sprintf("[0:v]split=%d", len(levels))

	for i := range levels {
		fc += fmt.Sprintf("[v%d]", i)
	}
	fc += ";"

	for i, h := range levels {
		if svc.VideoEncoder == "h264_nvenc" {
			fc += fmt.Sprintf("[v%d]scale_cuda=-2:%d[v%do];", i, h, i)
		} else if svc.VideoEncoder == "h264_qsv" {
			fc += fmt.Sprintf("[v%d]vpp_qsv=w=-2:%d[v%do];", i, h, i)
		} else {
			fc += fmt.Sprintf("[v%d]scale=-2:%d[v%do];", i, h, i)
		}
	}

	return strings.TrimSuffix(fc, ";")
}

func buildResolutionHeightLadder(maxresolutionHeight, n int) []int {
	var standardHeights = []int{
		2160,
		1440,
		1080,
		720,
		480,
		360,
		240,
	}
	out := []int{}

	for _, h := range standardHeights {
		if h <= maxresolutionHeight {
			out = append(out, h)
		}
		if len(out) == n {
			break
		}
	}

	return out
}

func getResolutionHeightBitrate(resolutionHeight int) string {
	switch resolutionHeight {
	case 2160:
		return "12000k"
	case 1440:
		return "6000k"
	case 1080:
		return "3000k"
	case 720:
		return "1800k"
	case 480:
		return "1000k"
	case 360:
		return "700k"
	case 240:
		return "400k"
	default:
		return "1000k"
	}
}

func (svc *Service) BuildffmpegArgs(
	filePath,
	outputDir string,
	duration float64,
	maxResHeight int,
) []string {
	videoHeightLevels := buildResolutionHeightLadder(maxResHeight, 3)
	args := []string{
		"-y",
	}

	if svc.VideoEncoder == "h264_nvenc" {
		args = append(args,
			"-hwaccel", "cuda",
			"-hwaccel_output_format", "cuda",
		)
	} else if svc.VideoEncoder == "h264_qsv" {
		args = append(args,
			"-hwaccel", "qsv",
			"-hwaccel_output_format", "qsv",
		)
	}

	args = append(args,
		"-i", filePath,
		"-filter_complex", svc.buildFilterComplex(videoHeightLevels),
	)

	for i, h := range videoHeightLevels {
		args = append(args,
			"-map", fmt.Sprintf("[v%do]", i),
			"-map", "0:a?",
		)

		// video encoder
		args = append(args,
			fmt.Sprintf("-c:v:%d", i), svc.VideoEncoder,
			fmt.Sprintf("-b:v:%d", i), getResolutionHeightBitrate(h),
		)

		// audio
		args = append(args,
			fmt.Sprintf("-c:a:%d", i), "aac",
			fmt.Sprintf("-b:a:%d", i), "128k",
		)
	}

	// HLS settings
	args = append(args,
		"-f", "hls",
		"-hls_time", "6",
		"-hls_playlist_type", "vod",
		"-master_pl_name", "master.m3u8",
		"-hls_segment_filename", filepath.Join(outputDir, "%v/segment_%03d.ts"),
	)
	// var_stream_map
	var maps []string
	for i := range videoHeightLevels {
		maps = append(maps, fmt.Sprintf("v:%d,a:%d", i, i))
	}

	args = append(args,
		"-var_stream_map", strings.Join(maps, " "),
		filepath.Join(outputDir, "%v/index.m3u8"),
	)
	return args
}
