package maptiles

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"runtime"
	"strconv"
	"time"

	log "github.com/cihub/seelog"
	"tileserver/ligneous"
)

func init() {
	logger, _ := ligneous.InitLogger()
	log.UseLogger(logger)
}

// Handles HTTP requests for map tiles, caching any produced tiles
// in an MBtiles 1.2 compatible sqlite db.
type TileServerSqlite struct {
	engine    string
	m         *TileDbSqlite3
	lmp       *LayerMultiplex
	TmsSchema bool
	startTime time.Time
}

func NewTileServerSqlite(cacheFile string) *TileServerSqlite {
	t := TileServerSqlite{}
	t.lmp = NewLayerMultiplex()
	t.m = NewTileDbSqlite(cacheFile)
	// t.m = NewTileDbPostgresql(cacheFile)
	t.startTime = time.Now()
	return &t
}

func (self *TileServerSqlite) AddMapnikLayer(layerName string, stylesheet string) {
	self.lmp.AddRenderer(layerName, stylesheet)
}

func (self *TileServerSqlite) ServeTileRequest(w http.ResponseWriter, r *http.Request, tc TileCoord) {
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

	// upgrade go1.7
	//log.Info(len(r.Cancel))
	//log.Info(r.Context())

	w.Header().Set("Content-Type", "image/png")
	w.WriteHeader(http.StatusOK)

	_, err := w.Write(result.BlobPNG)
	if err != nil {
		log.Error(err)
	}
	if needsInsert {
		self.m.InsertQueue() <- result // insert newly rendered tile into cache db
	}

}

func (self *TileServerSqlite) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	start := time.Now()

	if "/" == r.URL.Path {
		log.Info(fmt.Sprintf("%v %v %v ", r.RemoteAddr, r.URL.Path, time.Since(start)))
		self.IndexHandler(w, r)
		return
	} else if "/ping" == r.URL.Path {
		log.Info(fmt.Sprintf("%v %v %v ", r.RemoteAddr, r.URL.Path, time.Since(start)))
		self.PingHandler(w, r)
		return
	} else if "/server" == r.URL.Path {
		log.Info(fmt.Sprintf("%v %v %v ", r.RemoteAddr, r.URL.Path, time.Since(start)))
		self.ServerHandler(w, r)
		return
	} else if "/metadata" == r.URL.Path {
		log.Info(fmt.Sprintf("%v %v %v ", r.RemoteAddr, r.URL.Path, time.Since(start)))
		self.MetadataHandler(w, r)
		return
	} else if "/tilelayers" == r.URL.Path {
		log.Info(fmt.Sprintf("%v %v %v ", r.RemoteAddr, r.URL.Path, time.Since(start)))
		self.TileLayersHandler(w, r)
		return
	}

	var pathRegex = regexp.MustCompile(`/([A-Za-z0-9]+)/([0-9]+)/([0-9]+)/([0-9]+)\.png`)
	path := pathRegex.FindStringSubmatch(r.URL.Path)

	if path == nil {
		log.Info(fmt.Sprintf("%v %v %v ", r.RemoteAddr, r.URL.Path, time.Since(start)))
		self.RequestErrorHandler(w, r)
		return
	}

	l := path[1]
	z, _ := strconv.ParseUint(path[2], 10, 64)
	x, _ := strconv.ParseUint(path[3], 10, 64)
	y, _ := strconv.ParseUint(path[4], 10, 64)

	self.ServeTileRequest(w, r, TileCoord{x, y, z, self.TmsSchema, l})

	log.Info(fmt.Sprintf("%v %v %v ", r.RemoteAddr, r.URL.Path, time.Since(start)))
}

func (self *TileServerSqlite) RequestErrorHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	response := make(map[string]interface{})
	response["status"] = "error"
	result := make(map[string]interface{})
	result["message"] = "Expecting /{datasource}/{z}/{x}/{y}.png"
	response["data"] = result
	js, err := json.Marshal(response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(js)
}

func (self *TileServerSqlite) IndexHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	response := make(map[string]interface{})
	response["status"] = "ok"
	result := make(map[string]interface{})
	result["message"] = "Hello there ladies and gentlemen!"
	response["data"] = result
	js, err := json.Marshal(response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(js)
}

func (self *TileServerSqlite) MetadataHandler(w http.ResponseWriter, r *http.Request) {
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
func (self *TileServerSqlite) PingHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	response := make(map[string]interface{})
	response["status"] = "ok"
	result := make(map[string]interface{})
	result["result"] = "Pong"
	response["data"] = result
	js, err := json.Marshal(response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(js)
}

// ServerProfile returns basic server stats
func (self *TileServerSqlite) ServerHandler(w http.ResponseWriter, r *http.Request) {
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

func (self *TileServerSqlite) TileLayersHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	var keys []string
	for k := range self.lmp.layerChans {
		keys = append(keys, k)
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
