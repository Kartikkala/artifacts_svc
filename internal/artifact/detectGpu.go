package artifact

import (
	"os/exec"
	"runtime"
	"strings"
)

type GPUVendor string

const (
	VendorNvidia GPUVendor = "nvidia"
	VendorAMD    GPUVendor = "amd"
	VendorIntel  GPUVendor = "intel"
	VendorNone   GPUVendor = "none"
)

func DetectGPUVendor() GPUVendor {
	if runtime.GOOS == "windows" {
		return detectWindows()
	} else if runtime.GOOS == "linux" {
		return detectLinux()
	}
	return VendorNone
}

func encoderForVendor(v GPUVendor) string {
	switch v {
	case VendorNvidia:
		return "h264_nvenc"
	case VendorIntel:
		return "h264_qsv"
	case VendorAMD:
		return "h264_vaapi"
	default:
		return "libx264"
	}
}

func detectWindows() GPUVendor {
	cmd := exec.Command("wmic", "path", "win32_videocontroller", "get", "PNPDeviceID")
	out, err := cmd.Output()
	if err != nil {
		return VendorNone
	}

	output := strings.ToUpper(string(out))
	if strings.Contains(output, "VEN_10DE") {
		return VendorNvidia
	}
	if strings.Contains(output, "VEN_1002") {
		return VendorAMD
	}
	if strings.Contains(output, "VEN_8086") {
		return VendorIntel
	}
	return VendorNone
}

func detectLinux() GPUVendor {
	cmd := exec.Command("lspci", "-nn")
	out, err := cmd.Output()
	if err != nil {
		return VendorNone
	}

	output := strings.ToLower(string(out))

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "vga compatible controller") || strings.Contains(line, "3d controller") {
			if strings.Contains(line, "10de:") {
				return VendorNvidia
			}
			if strings.Contains(line, "1002:") {
				return VendorAMD
			}
			if strings.Contains(line, "8086:") {
				return VendorIntel
			}
		}
	}
	return VendorNone
}
