package maptiles

import (
	"log"
	"net/http"
	"regexp"
	"strconv"
	"encoding/json"
	"time"
	"runtime"
)

// TODO serve list of registered layers per HTTP (preferably leafletjs-compatible js-array)

// Handles HTTP requests for map tiles, caching any produced tiles
// in an MBtiles 1.2 compatible sqlite db.
type TileServerPostgres struct {
	engine string
	m         *TileDbPostgresql
	lmp       *LayerMultiplex
	TmsSchema bool
	startTime time.Time
}

func NewTileServerPostgres(cacheFile string) *TileServerPostgres {
	t := TileServerPostgres{}
	t.lmp = NewLayerMultiplex()
	// t.m = NewTileDbSqlite(cacheFile)
	t.m = NewTileDbPostgresql(cacheFile)
	t.startTime = time.Now()
	return &t
}

func (self *TileServerPostgres) AddMapnikLayer(layerName string, stylesheet string) {
	self.lmp.AddRenderer(layerName, stylesheet)
}

func (self *TileServerPostgres) ServeTileRequest(w http.ResponseWriter, r *http.Request, tc TileCoord) {
	// log.Println(r.RemoteAddr, tc)
	ch := make(chan TileFetchResult)

	tr := TileFetchRequest{tc, ch}
	self.m.RequestQueue() <- tr

	result := <-ch
	needsInsert := false

	if result.BlobPNG == nil {
		// Tile was not provided by DB, so submit the tile request to the renderer
		self.lmp.SubmitRequest(tr)
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
		self.m.InsertQueue() <- result // insert newly rendered tile into cache db
	}
}

func (self *TileServerPostgres) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	log.Println(r.RemoteAddr, r.URL.Path)

	if "/" == r.URL.Path {
		self.IndexHandler(w, r)
		return
	} else if "/ping" == r.URL.Path {
		self.PingHandler(w, r)
		return
	} else if "/server" == r.URL.Path {
		self.ServerHandler(w, r)
		return
	} else if "/metadata" == r.URL.Path {
		self.MetadataHandler(w, r)
		return
	}else if "/tilelayers" == r.URL.Path {
		self.TileLayersHandler(w, r)
		return
	}

	var pathRegex = regexp.MustCompile(`/([A-Za-z0-9]+)/([0-9]+)/([0-9]+)/([0-9]+)\.png`)
	path := pathRegex.FindStringSubmatch(r.URL.Path)

	if path == nil {
		http.NotFound(w, r)
		return
	}

	l := path[1]
	z, _ := strconv.ParseUint(path[2], 10, 64)
	x, _ := strconv.ParseUint(path[3], 10, 64)
	y, _ := strconv.ParseUint(path[4], 10, 64)

	self.ServeTileRequest(w, r, TileCoord{x, y, z, self.TmsSchema, l})
}


func (self *TileServerPostgres) IndexHandler(w http.ResponseWriter, r *http.Request) {
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

func (self *TileServerPostgres) MetadataHandler(w http.ResponseWriter, r *http.Request) {
	// todo: include layer
	metadata := self.m.MetaDataHandler()
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
func (self *TileServerPostgres) PingHandler(w http.ResponseWriter, r *http.Request) {
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
func (self *TileServerPostgres) ServerHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	var data map[string]interface{}
	data = make(map[string]interface{})
	data["registered"] = self.startTime.UTC()
	data["uptime"] = time.Since(self.startTime).Seconds()
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


func (self *TileServerPostgres) TileLayersHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	var keys []string
	for k := range self.lmp.layerChans {
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

