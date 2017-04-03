package maptiles

import (
	"fmt"
	"regexp"
	"strconv"
)

// ParseTileUrl validates tile url and returns layer, x, y and z parameters
// for tile lookup.
func ParseTileUrl(url_path string) (string, []uint64, error) {
	var pathRegex = regexp.MustCompile(`/([A-Za-z0-9]+)/([0-9]+)/([0-9]+)/([0-9]+)\.png`)
	path := pathRegex.FindStringSubmatch(url_path)
	if nil == path {
		return "", []uint64{}, fmt.Errorf("Unable to parse url")
	}
	layer, xyz := GetTileUrlParts(path)
	return layer, xyz, nil
}

// GetTileUrlParts gets layer, x, y and z parameters from string list.
func GetTileUrlParts(path []string) (string, []uint64) {
	l := path[1]
	z, _ := strconv.ParseUint(path[2], 10, 64)
	x, _ := strconv.ParseUint(path[3], 10, 64)
	y, _ := strconv.ParseUint(path[4], 10, 64)
	return l, []uint64{x, y, z}
}
