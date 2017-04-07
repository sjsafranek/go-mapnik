package maptiles

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

import "mapnik"

// ProxyClient http client for server proxy tile layers.
var ProxyClient = &http.Client{
	Timeout: time.Second * 30,
}

// TileCoord struct for tile requests.
type TileCoord struct {
	X, Y, Zoom uint64
	Tms        bool
	Layer      string
}

// OSMFilename formats png filename.
func (c TileCoord) OSMFilename() string {
	return fmt.Sprintf("%d/%d/%d.png", c.Zoom, c.X, c.Y)
}

// TileFetchResult struct for tile result.
type TileFetchResult struct {
	Coord   TileCoord
	BlobPNG []byte
}

// TileFetchRequest struct for tile request.
type TileFetchRequest struct {
	Coord   TileCoord
	OutChan chan<- TileFetchResult
}

// setTMS sets Tmns to signal server proxy tile layer.
func (c *TileCoord) setTMS(tms bool) {
	if c.Tms != tms {
		c.Y = (1 << c.Zoom) - c.Y - 1
		c.Tms = tms
	}
}

// NewTileRendererChan creates channel for tile rendering
func NewTileRendererChan(stylesheet string) chan<- TileFetchRequest {
	c := make(chan TileFetchRequest)

	go func(requestChan <-chan TileFetchRequest) {
		var err error
		t := NewTileRenderer(stylesheet)
		for request := range requestChan {
			result := TileFetchResult{request.Coord, nil}
			result.BlobPNG, err = t.RenderTile(request.Coord)
			if err != nil {
				Ligneous.Error("Error while rendering", request.Coord, ":", err.Error())
				result.BlobPNG = nil
			}
			request.OutChan <- result
		}
	}(c)

	return c
}

// TileRenderer renders images as Web Mercator tiles.
type TileRenderer struct {
	m     *mapnik.Map
	mp    mapnik.Projection
	proxy bool
	s     string
}

// NewTileRenderer creates TileRenderer struct.
func NewTileRenderer(stylesheet string) *TileRenderer {
	t := new(TileRenderer)
	var err error
	if err != nil {
		Ligneous.Critical(err)
	}
	t.m = mapnik.NewMap(256, 256)
	t.m.Load(stylesheet)
	t.mp = t.m.Projection()

	if strings.Contains(stylesheet, ".xml") {
		t.proxy = false
		t.s = stylesheet
	} else if strings.Contains(stylesheet, "http") && strings.Contains(stylesheet, "{z}/{x}/{y}") {
		t.proxy = true
		t.s = stylesheet
	} else if strings.Contains(stylesheet, "http") && strings.Contains(stylesheet, "{z}/{y}/{x}") {
		t.proxy = true
		t.s = stylesheet
	}

	return t
}

// RenderTile renders map tile.
func (t *TileRenderer) RenderTile(c TileCoord) ([]byte, error) {
	c.setTMS(false)
	if t.proxy {
		return t.HttpGetTileZXY(c.Zoom, c.X, c.Y)
	} else {
		return t.RenderTileZXY(c.Zoom, c.X, c.Y)
	}
}

// RenderTileZXY renders map tile.
// Render a tile with coordinates in Google tile format.
// Most upper left tile is always 0,0. Method is not thread-safe,
// so wrap with a mutex when accessing the same renderer by multiple
// threads or setup multiple goroutinesand communicate with channels,
// see NewTileRendererChan.
func (t *TileRenderer) RenderTileZXY(zoom, x, y uint64) ([]byte, error) {
	// Calculate pixel positions of bottom left & top right
	p0 := [2]float64{float64(x) * 256, (float64(y) + 1) * 256}
	p1 := [2]float64{(float64(x) + 1) * 256, float64(y) * 256}

	// Convert to LatLong(EPSG:4326)
	l0 := fromPixelToLL(p0, zoom)
	l1 := fromPixelToLL(p1, zoom)

	// Convert to map projection (e.g. mercartor co-ords EPSG:3857)
	c0 := t.mp.Forward(mapnik.Coord{l0[0], l0[1]})
	c1 := t.mp.Forward(mapnik.Coord{l1[0], l1[1]})

	// Bounding box for the Tile
	t.m.Resize(256, 256)
	t.m.ZoomToMinMax(c0.X, c0.Y, c1.X, c1.Y)
	t.m.SetBufferSize(128)

	blob, err := t.m.RenderToMemoryPng()

	Ligneous.Trace(fmt.Sprintf("RENDER BLOB %v %v %v %v", t.s, zoom, x, y))

	return blob, err
}

// subDomain selects random sub domain for proxy tile server.
func (t *TileRenderer) subDomain() string {
	subs := []string{"a", "b", "c"}
	n := RandomIntBetween(0, 3)
	return subs[n]
}

// HttpGetTileZXY gets map tile another tile server.
func (t *TileRenderer) HttpGetTileZXY(zoom, x, y uint64) ([]byte, error) {
	tileUrl := strings.Replace(t.s, "{z}", fmt.Sprintf("%v", zoom), -1)
	tileUrl = strings.Replace(tileUrl, "{x}", fmt.Sprintf("%v", x), -1)
	tileUrl = strings.Replace(tileUrl, "{y}", fmt.Sprintf("%v", y), -1)
	tileUrl = strings.Replace(tileUrl, "{s}", t.subDomain(), -1)

	// Retry attempts -- 5
	attempt := 0
	for {
		//resp, err := ProxyClient.Get(tileUrl)
		//.start :: tile request
		req, err := http.NewRequest("GET", tileUrl, nil)
		if err != nil {
			Ligneous.Error(err)
		}
		//req.Header.Set("User-Agent", "Golang_TileServer/1.2")
		// Look like a web browser running leaflet
		req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/54.0.2840.71 Safari/537.36")
		resp, err := ProxyClient.Do(req)
		//.end

		if nil == err {

			blob, err := ioutil.ReadAll(resp.Body)
			defer resp.Body.Close()
			if nil != err {
				return []byte{}, err
			}

			Ligneous.Trace(fmt.Sprintf("PROXY GET %v %v", tileUrl, resp.StatusCode))

			if 200 != resp.StatusCode {
				err := errors.New("Request error: " + string(blob))
				return []byte{}, err
			}

			return blob, err
		}

		attempt++
		if attempt > 4 {
			return []byte{}, err
		}
	}
}
