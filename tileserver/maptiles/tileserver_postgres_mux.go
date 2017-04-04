package maptiles

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

// TileServerPostgresMux PostgresSQL tile server.
// Handles HTTP requests for map tiles, caching any produced tiles
// in an MBtiles 1.2 compatible sqlite db.
type TileServerPostgresMux struct {
	engine    string
	m         *TileDbPostgresql
	lmp       *LayerMultiplex
	TmsSchema bool
	startTime time.Time
	Router    *mux.Router
}

// NewTileServerPostgresMux creates TileServerPostgresMux object.
func NewTileServerPostgresMux(cacheFile string) *TileServerPostgresMux {
	t := TileServerPostgresMux{}
	t.lmp = NewLayerMultiplex()
	t.m = NewTileDbPostgresql(cacheFile)
	t.startTime = time.Now()

	t.Router = mux.NewRouter()
	t.Router.HandleFunc("/ping", t.PingHandler).Methods("GET")
	t.Router.HandleFunc("/server", t.ServerProfileHandler).Methods("GET")
	//t.Router.HandleFunc("/{lyr}/metadata", t.MetadataHandler).Methods("GET")
	t.Router.HandleFunc("/tilelayers", t.TileLayersHandler).Methods("GET")
	t.Router.HandleFunc("/", t.IndexHandler).Methods("GET")
	t.Router.HandleFunc("/tms/1.0", t.TMSTileMaps).Methods("GET")
	t.Router.HandleFunc("/tms/1.0/{lyr}", t.TMSTileMap).Methods("GET")
	t.Router.HandleFunc("/tms/1.0/{lyr}/{z:[0-9]+}", t.TMSErrorTile).Methods("GET")
	t.Router.HandleFunc("/tms/1.0/{lyr}/{z:[0-9]+}/{x:[0-9]+}", t.TMSErrorTile).Methods("GET")
	t.Router.HandleFunc("/tms/1.0/{lyr}/{z:[0-9]+}/{x:[0-9]+}/{y:[0-9]+}.png", t.ServeTileRequest).Methods("GET")

	return &t
}

// AddMapnikLayer adds mapnik layer to server.
func (self *TileServerPostgresMux) AddMapnikLayer(layerName string, stylesheet string) {
	self.m.AddLayerMetadata(layerName, stylesheet)
	self.lmp.AddRenderer(layerName, stylesheet)
}

// ServeTileRequest serves tile request.
func (self *TileServerPostgresMux) ServeTileRequest(w http.ResponseWriter, r *http.Request) {

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

// IndexHandler for server.
func (self *TileServerPostgresMux) IndexHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	TMSRootHandler(w, r)
	Ligneous.Info(fmt.Sprintf("%v %v %v ", r.RemoteAddr, r.URL.Path, time.Since(start)))
}

// TMSTileMaps lists available TileMaps
func (self *TileServerPostgresMux) TMSTileMaps(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	var layers []string
	for k := range self.lmp.layerChans {
		layers = append(layers, k)
	}
	TMSTileMaps(layers, w, r)
	Ligneous.Info(fmt.Sprintf("%v %v %v ", r.RemoteAddr, r.URL.Path, time.Since(start)))
}

// TMSTileMap shows list of TileSets for layer
func (self *TileServerPostgresMux) TMSTileMap(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	vars := mux.Vars(r)
	lyr := vars["lyr"]
	metadata := self.m.MetaDataHandler(lyr)
	if _, ok := self.lmp.layerChans[lyr]; !ok {
		http.Error(w, "layer not found", http.StatusNotFound)
	} else {
		TMSTileMap(lyr, metadata["source"], w, r)
	}
	Ligneous.Info(fmt.Sprintf("%v %v %v ", r.RemoteAddr, r.URL.Path, time.Since(start)))
}

// TMSErrorTile returns error response
func (self *TileServerPostgresMux) TMSErrorTile(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	http.Error(w, "Expecting /{layer}/{z}/{x}/{y}.png", http.StatusBadRequest)
	Ligneous.Info(fmt.Sprintf("%v %v %v ", r.RemoteAddr, r.URL.Path, time.Since(start)))
}

// // MetadataHandler for tile server.
// func (self *TileServerPostgresMux) MetadataHandler(w http.ResponseWriter, r *http.Request) {
// 	start := time.Now()
// 	vars := mux.Vars(r)
// 	lyr := vars["lyr"]
// 	metadata := self.m.MetaDataHandler(lyr)
// 	response := make(map[string]interface{})
// 	response["status"] = "ok"
// 	response["data"] = metadata
// 	SendJsonResponseFromInterface(w, r, response)
// 	Ligneous.Info(fmt.Sprintf("%v %v %v ", r.RemoteAddr, r.URL.Path, time.Since(start)))
// }

// PingHandler provides an api route for server health check
func (self *TileServerPostgresMux) PingHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	PingHandler(w, r)
	Ligneous.Info(fmt.Sprintf("%v %v %v ", r.RemoteAddr, r.URL.Path, time.Since(start)))
}

// ServerProfileHandler returns basic server stats.
func (self *TileServerPostgresMux) ServerProfileHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	ServerProfileHandler(self.startTime, w, r)
	Ligneous.Info(fmt.Sprintf("%v %v %v ", r.RemoteAddr, r.URL.Path, time.Since(start)))
}

// TileLayersHandler returns list of tiles.
func (self *TileServerPostgresMux) TileLayersHandler(w http.ResponseWriter, r *http.Request) {
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
