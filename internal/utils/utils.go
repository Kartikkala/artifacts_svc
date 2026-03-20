package utils

import (
	"fmt"
	"net/url"
	"path"
	"regexp"
	"strings"
)

var HTTP_HOSTNAME_REGEXP = regexp.MustCompile(
	`^(([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9-]*[a-zA-Z0-9])\.)*([A-Za-z0-9]|[A-Za-z0-9][A-Za-z0-9-]*[A-Za-z0-9])$`,
)

var ENDPOINT_REGEXP = regexp.MustCompile(
	`^(/[a-zA-Z0-9]+)+$`,
)

func isValidAddress(
	h string,
	protocol Protocol,
) bool {
	if protocol == UNIX {
		return path.IsAbs(h)
	}
	return HTTP_HOSTNAME_REGEXP.Match([]byte(h))
}

func isValidEndpoint(
	e string,
) bool {
	return ENDPOINT_REGEXP.Match([]byte(e))
}

func ConstructURL(
	protocol Protocol,
	address string,
	port uint16,
	endpoint string,
) (*url.URL, error) {
	if !isValidAddress(address, protocol) {
		return nil, fmt.Errorf("invalid address\n")
	}
	if !isValidEndpoint(endpoint) {
		return nil, fmt.Errorf("invalid characters in endpoint\n")
	}

	if protocol == UNIX {
		return &url.URL{
			Scheme: "unix",
			Path:   address,
		}, nil
	}
	endpoint = strings.TrimSpace(endpoint)
	endpoint = strings.Trim(endpoint, "/")

	return &url.URL{
		Scheme: string(protocol),
		Host:   fmt.Sprintf("%s:%d", address, port),
		Path:   "/" + endpoint,
	}, nil
}
