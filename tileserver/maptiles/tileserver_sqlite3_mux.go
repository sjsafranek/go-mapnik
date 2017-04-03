package maptiles

import (
	"fmt"
	"net/http"
	"runtime"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

// TileServerSqliteMux SQLite3 tile server.
// Handles HTTP requests for map tiles, caching any produced tiles
// in an MBtiles 1.2 compatible sqlite db.
type TileServerSqliteMux struct {
	engine    string
	m         *TileDbSqlite3
	lmp       *LayerMultiplex
	TmsSchema bool
	startTime time.Time
	Router    *mux.Router
}

// NewTileServerSqliteMux creates TileServerSqliteMux object.
func NewTileServerSqliteMux(cacheFile string) *TileServerSqliteMux {
	t := TileServerSqliteMux{}
	t.lmp = NewLayerMultiplex()
	t.m = NewTileDbSqlite(cacheFile)
	t.startTime = time.Now()

	t.Router = mux.NewRouter()
	t.Router.HandleFunc("/", t.IndexHandler).Methods("GET")
	t.Router.HandleFunc("/ping", t.PingHandler).Methods("GET")
	t.Router.HandleFunc("/server", t.ServerProfileHandler).Methods("GET")
	t.Router.HandleFunc("/metadata", t.MetadataHandler).Methods("GET")
	t.Router.HandleFunc("/tilelayers", t.TileLayersHandler).Methods("GET")
	t.Router.HandleFunc("/{lyr}/{z:[0-9]+}/{x:[0-9]+}/{y:[0-9]+}.png", t.ServeTileRequest).Methods("GET")

	return &t
}

// AddMapnikLayer adds mapnik layer to server.
func (self *TileServerSqliteMux) AddMapnikLayer(layerName string, stylesheet string) {
	self.lmp.AddRenderer(layerName, stylesheet)
}

// ServeTileRequest serves tile request.
func (self *TileServerSqliteMux) ServeTileRequest(w http.ResponseWriter, r *http.Request) {

	start := time.Now()

	vars := mux.Vars(r)
	lyr := vars["lyr"]
	z, _ := strconv.ParseUint(vars["z"], 10, 64)
	x, _ := strconv.ParseUint(vars["x"], 10, 64)
	y, _ := strconv.ParseUint(vars["y"], 10, 64)

	tc := TileCoord{x, y, z, self.TmsSchema, lyr}

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
	w.WriteHeader(http.StatusOK)

	_, err := w.Write(result.BlobPNG)
	if err != nil {
		Ligneous.Error(err)
	}
	if needsInsert {
		self.m.InsertQueue() <- result // insert newly rendered tile into cache db
	}

	Ligneous.Info(fmt.Sprintf("%v %v %v ", r.RemoteAddr, r.URL.Path, time.Since(start)))
}

// RequestErrorHandler handles error response.
func (self *TileServerSqliteMux) RequestErrorHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	response := make(map[string]interface{})
	response["status"] = "error"
	result := make(map[string]interface{})
	result["message"] = "Expecting /{datasource}/{z}/{x}/{y}.png"
	response["data"] = result
	SendJsonResponseFromInterface(w, r, response)
	Ligneous.Info(fmt.Sprintf("%v %v %v ", r.RemoteAddr, r.URL.Path, time.Since(start)))
}

// IndexHandler for server.
func (self *TileServerSqliteMux) IndexHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	fmt.Fprintf(w, "MapnikServer\nHello there ladies and gentlemen!")
	Ligneous.Info(fmt.Sprintf("%v %v %v ", r.RemoteAddr, r.URL.Path, time.Since(start)))
}

// MetadataHandler for tile server.
func (self *TileServerSqliteMux) MetadataHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	metadata := self.m.MetaDataHandler()
	response := make(map[string]interface{})
	response["status"] = "ok"
	response["data"] = metadata
	SendJsonResponseFromInterface(w, r, response)
	Ligneous.Info(fmt.Sprintf("%v %v %v ", r.RemoteAddr, r.URL.Path, time.Since(start)))
}

// PingHandler provides an api route for server health check
func (self *TileServerSqliteMux) PingHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	response := make(map[string]interface{})
	response["status"] = "ok"
	result := make(map[string]interface{})
	result["result"] = "Pong"
	response["data"] = result
	SendJsonResponseFromInterface(w, r, response)
	Ligneous.Info(fmt.Sprintf("%v %v %v ", r.RemoteAddr, r.URL.Path, time.Since(start)))
}

// ServerProfileHandler returns basic server stats.
func (self *TileServerSqliteMux) ServerProfileHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	var data map[string]interface{}
	data = make(map[string]interface{})
	data["registered"] = self.startTime.UTC()
	data["uptime"] = time.Since(self.startTime).Seconds()
	data["num_cores"] = runtime.NumCPU()
	response := make(map[string]interface{})
	response["status"] = "ok"
	response["data"] = data
	SendJsonResponseFromInterface(w, r, response)
	Ligneous.Info(fmt.Sprintf("%v %v %v ", r.RemoteAddr, r.URL.Path, time.Since(start)))
}

// TileLayersHandler returns list of tiles.
func (self *TileServerSqliteMux) TileLayersHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	var keys []string
	for k := range self.lmp.layerChans {
		keys = append(keys, k)
	}
	var response map[string]interface{}
	response = make(map[string]interface{})
	response["status"] = "ok"
	response["data"] = keys
	SendJsonResponseFromInterface(w, r, response)
	Ligneous.Info(fmt.Sprintf("%v %v %v ", r.RemoteAddr, r.URL.Path, time.Since(start)))
}
