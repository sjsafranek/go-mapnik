package maptiles

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"strconv"
	"time"

	log "github.com/cihub/seelog"
	"github.com/gorilla/mux"
)

import "ligneous"

func init() {
	logger, _ := ligneous.InitLogger("SQLite3 Mux")
	log.UseLogger(logger)
}

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

func NewTileServerSqliteMux(cacheFile string) *TileServerSqliteMux {
	t := TileServerSqliteMux{}
	t.lmp = NewLayerMultiplex()
	t.m = NewTileDbSqlite(cacheFile)
	// t.m = NewTileDbPostgresql(cacheFile)
	t.startTime = time.Now()

	t.Router = mux.NewRouter()
	t.Router.HandleFunc("/", t.IndexHandler).Methods("GET")
	t.Router.HandleFunc("/ping", t.PingHandler).Methods("GET")
	t.Router.HandleFunc("/server", t.ServerHandler).Methods("GET")
	t.Router.HandleFunc("/metadata", t.MetadataHandler).Methods("GET")
	t.Router.HandleFunc("/tilelayers", t.TileLayersHandler).Methods("GET")
	t.Router.HandleFunc("/{lyr}/{z:[0-9]+}/{x:[0-9]+}/{y:[0-9]+}.png", t.ServeTileRequest).Methods("GET")

	return &t
}

func (self *TileServerSqliteMux) AddMapnikLayer(layerName string, stylesheet string) {
	self.lmp.AddRenderer(layerName, stylesheet)
}

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

	log.Info(fmt.Sprintf("%v %v %v ", r.RemoteAddr, r.URL.Path, time.Since(start)))

}

func (self *TileServerSqliteMux) RequestErrorHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
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
	log.Info(fmt.Sprintf("%v %v %v ", r.RemoteAddr, r.URL.Path, time.Since(start)))
}

func (self *TileServerSqliteMux) IndexHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
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
	log.Info(fmt.Sprintf("%v %v %v ", r.RemoteAddr, r.URL.Path, time.Since(start)))
}

func (self *TileServerSqliteMux) MetadataHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
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
	log.Info(fmt.Sprintf("%v %v %v ", r.RemoteAddr, r.URL.Path, time.Since(start)))
}

// PingHandler provides an api route for server health check
func (self *TileServerSqliteMux) PingHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
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
	log.Info(fmt.Sprintf("%v %v %v ", r.RemoteAddr, r.URL.Path, time.Since(start)))
}

// ServerProfile returns basic server stats
func (self *TileServerSqliteMux) ServerHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
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
	log.Info(fmt.Sprintf("%v %v %v ", r.RemoteAddr, r.URL.Path, time.Since(start)))
}

func (self *TileServerSqliteMux) TileLayersHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
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
	log.Info(fmt.Sprintf("%v %v %v ", r.RemoteAddr, r.URL.Path, time.Since(start)))
}
