package maptiles

import (
	"fmt"
	"regexp"
)

func ParseTileUrl(url_path string) ([]string, error) {
	var pathRegex = regexp.MustCompile(`/([A-Za-z0-9]+)/([0-9]+)/([0-9]+)/([0-9]+)\.png`)
	path := pathRegex.FindStringSubmatch(url_path)
	if nil == path {
		return path, fmt.Errorf("Unable to parse url")
	}
	return path, nil
}
