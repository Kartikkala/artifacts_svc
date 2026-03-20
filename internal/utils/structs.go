package utils

type Protocol string

const (
	HTTP  Protocol = "http"
	HTTPS Protocol = "https"
	UNIX  Protocol = "unix"
)
