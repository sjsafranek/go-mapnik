package maptiles

import (
	"fmt"
	"strings"
	"io/ioutil"
	"net/http"
	"errors"
	"time"
	"math/rand"

	"tileserver/mapnik"
	"tileserver/ligneous"
	
	log "github.com/cihub/seelog"
)


func init() {
	logger, _ := ligneous.InitLogger()
	log.UseLogger(logger)
}


func randomNum(min, max int) int {
    rand.Seed(time.Now().Unix())
    return rand.Intn(max - min) + min
}


type TileCoord struct {
	X, Y, Zoom uint64
	Tms        bool
	Layer      string
}

func (c TileCoord) OSMFilename() string {
	return fmt.Sprintf("%d/%d/%d.png", c.Zoom, c.X, c.Y)
}

type TileFetchResult struct {
	Coord   TileCoord
	BlobPNG []byte
}

type TileFetchRequest struct {
	Coord   TileCoord
	OutChan chan<- TileFetchResult
}

func (c *TileCoord) setTMS(tms bool) {
	if c.Tms != tms {
		c.Y = (1 << c.Zoom) - c.Y - 1
		c.Tms = tms
	}
}

func NewTileRendererChan(stylesheet string) chan<- TileFetchRequest {
	c := make(chan TileFetchRequest)

	go func(requestChan <-chan TileFetchRequest) {
		var err error
		t := NewTileRenderer(stylesheet)
		for request := range requestChan {
			result := TileFetchResult{request.Coord, nil}
			result.BlobPNG, err = t.RenderTile(request.Coord)
			if err != nil {
				log.Error("Error while rendering", request.Coord, ":", err.Error())
				result.BlobPNG = nil
			}
			request.OutChan <- result
		}
	}(c)

	return c
}

// Renders images as Web Mercator tiles
type TileRenderer struct {
	m  *mapnik.Map
	mp mapnik.Projection
	proxy bool
	s string
}

func NewTileRenderer(stylesheet string) *TileRenderer {
	t := new(TileRenderer)
	var err error
	if err != nil {
		log.Critical(err)
	}
	t.m = mapnik.NewMap(256, 256)
	t.m.Load(stylesheet)
	t.mp = t.m.Projection()

	if strings.Contains(stylesheet, ".xml") {
		t.proxy = false
		t.s = stylesheet
	} else if strings.Contains(stylesheet, "http") && strings.Contains(stylesheet, "{z}/{x}/{y}.png") {
		t.proxy = true
		t.s = stylesheet
	}

	return t
}

func (t *TileRenderer) RenderTile(c TileCoord) ([]byte, error) {
	c.setTMS(false)
	if t.proxy {
		return t.HttpGetTileZXY(c.Zoom, c.X, c.Y)
	} else {
		return t.RenderTileZXY(c.Zoom, c.X, c.Y)
	}
}

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
	
	log.Debug( fmt.Sprintf("RENDER BLOB %v %v %v %v", t.s zoom, x, y) )
	
	return blob, err
}

func (t *TileRenderer) subDomain() string {
	subs := []string{"a","b","c"}
	n := randomNum(0,3);
	return subs[n]
} 

func (t *TileRenderer) HttpGetTileZXY(zoom, x, y uint64) ([]byte, error) {
	tileUrl := strings.Replace(t.s, "{z}", fmt.Sprintf("%v", zoom), -1);
	tileUrl = strings.Replace(tileUrl, "{x}", fmt.Sprintf("%v", x), -1);
	tileUrl = strings.Replace(tileUrl, "{y}", fmt.Sprintf("%v", y), -1);
	tileUrl = strings.Replace(tileUrl, "{s}", t.subDomain(), -1);
	
	resp, err := http.Get(tileUrl)
	blob, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	
	log.Debug( fmt.Sprintf("PROXY GET %v %v", tileUrl, resp.StatusCode) )
	
	if 200 != resp.StatusCode {
		err := errors.New("Request error: "+ string(blob))
		return []byte{}, err
	}

	return blob, err
}
