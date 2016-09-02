package maptiles

import (
	"log"
	"net/http"
	"regexp"
	"strconv"
	// "strings"
	"encoding/json"
	"time"
	"runtime"
)

// TODO serve list of registered layers per HTTP (preferably leafletjs-compatible js-array)

// Handles HTTP requests for map tiles, caching any produced tiles
// in an MBtiles 1.2 compatible sqlite db.
type TileServer struct {
	m         *TileDb
	lmp       *LayerMultiplex
	TmsSchema bool
	startTime time.Time
}

func NewTileServer(cacheFile string) *TileServer {
	t := TileServer{}
	t.lmp = NewLayerMultiplex()
	t.m = NewTileDb(cacheFile)
	t.startTime = time.Now()

	return &t
}

func (t *TileServer) AddMapnikLayer(layerName string, stylesheet string) {
	t.lmp.AddRenderer(layerName, stylesheet)
}

var pathRegex = regexp.MustCompile(`/([A-Za-z0-9]+)/([0-9]+)/([0-9]+)/([0-9]+)\.png`)

func (t *TileServer) ServeTileRequest(w http.ResponseWriter, r *http.Request, tc TileCoord) {
	log.Println(r.RemoteAddr, tc)
	ch := make(chan TileFetchResult)

	tr := TileFetchRequest{tc, ch}
	t.m.RequestQueue() <- tr

	result := <-ch
	needsInsert := false

	if result.BlobPNG == nil {
		// Tile was not provided by DB, so submit the tile request to the renderer
		t.lmp.SubmitRequest(tr)
		result = <-ch
		if result.BlobPNG == nil {
			// The tile could not be rendered, now we need to bail out.
			http.NotFound(w, r)
			return
		}
		needsInsert = true
	}

	w.Header().Set("Content-Type", "image/png")
	_, err := w.Write(result.BlobPNG)
	if err != nil {
		log.Println(err)
	}
	if needsInsert {
		t.m.InsertQueue() <- result // insert newly rendered tile into cache db
	}
}

func (t *TileServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	log.Println(r.RemoteAddr, r.URL.Path)

	if "/" == r.URL.Path {
		t.IndexHandler(w, r)
		return
	} else if "/ping" == r.URL.Path {
		t.PingHandler(w, r)
		return
	} else if "/server" == r.URL.Path {
		t.ServerHandler(w, r)
		return
	} else if "/metadata" == r.URL.Path {
		t.MetadataHandler(w, r)
		return
	}else if "/tilelayers" == r.URL.Path {
		t.TileLayersHandler(w, r)
		return
	}

	path := pathRegex.FindStringSubmatch(r.URL.Path)

	if path == nil {
		http.NotFound(w, r)
		return
	}

	l := path[1]
	z, _ := strconv.ParseUint(path[2], 10, 64)
	x, _ := strconv.ParseUint(path[3], 10, 64)
	y, _ := strconv.ParseUint(path[4], 10, 64)

	t.ServeTileRequest(w, r, TileCoord{x, y, z, t.TmsSchema, l})
}


func (t *TileServer) IndexHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	response := `{"status":"ok","data":{"message":"Hello there ladies and gentlemen!"}}`
	js, err := json.Marshal(response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(js)
}

func (t *TileServer) MetadataHandler(w http.ResponseWriter, r *http.Request) {
	// todo: include layer
	metadata := t.m.MetaDataHandler()
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	response := make(map[string]interface{})
	response["status"] = "ok"
	response["data"] = metadata
	js, err := json.Marshal(response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(js)
}

// PingHandler provides an api route for server health check
func (t *TileServer) PingHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	data := `{"status": "success", "data": {"result": "pong"}}`
	js, err := json.Marshal(data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(js)
}

// ServerProfile returns basic server stats
func (t *TileServer) ServerHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	var data map[string]interface{}
	data = make(map[string]interface{})
	data["registered"] = t.startTime.UTC()
	data["uptime"] = time.Since(t.startTime).Seconds()
	data["num_cores"] = runtime.NumCPU()
	response := make(map[string]interface{})
	response["status"] = "ok"
	response["data"] = data
	// data["free_mem"] = runtime.MemStats()
	js, err := json.Marshal(response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(js)
}


func (t *TileServer) TileLayersHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	var keys []string
	for k := range t.lmp.layerChans {
		keys = append(keys,k)
	}
	var response map[string]interface{}
	response = make(map[string]interface{})
	response["status"] = "ok"
	response["data"] = keys
	js, err := json.Marshal(response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(js)
}

