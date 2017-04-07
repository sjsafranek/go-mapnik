package maptiles

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
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

	tilelayers, err := t.m.GetTileLayers()
	if nil != err {
		Ligneous.Critical(err)
		panic(err)
	}
	Ligneous.Debug(tilelayers)
	for i := range tilelayers {
		t.AddMapnikLayer(tilelayers[i]["name"], tilelayers[i]["source"])
	}

	t.startTime = time.Now()

	t.Router = mux.NewRouter()
	t.Router.HandleFunc("/api/v1/tilelayer/{lyr}", t.GetTileLayer).Methods("Get")
	t.Router.HandleFunc("/api/v1/tilelayer", t.NewTileLayer).Methods("POST")
	t.Router.HandleFunc("/api/v1/tilelayers", t.TileLayersHandler).Methods("GET")
	t.Router.HandleFunc("/ping", PingHandler).Methods("GET")
	t.Router.HandleFunc("/server", t.ServerProfileHandler).Methods("GET")
	t.Router.HandleFunc("/", TMSRootHandler).Methods("GET")
	t.Router.HandleFunc("/tms/1.0", t.TMSTileMaps).Methods("GET")
	t.Router.HandleFunc("/tms/1.0/{lyr}", t.TMSTileMap).Methods("GET")
	t.Router.HandleFunc("/tms/1.0/{lyr}/{z:[0-9]+}", TMSErrorTile).Methods("GET")
	t.Router.HandleFunc("/tms/1.0/{lyr}/{z:[0-9]+}/{x:[0-9]+}", TMSErrorTile).Methods("GET")
	t.Router.HandleFunc("/tms/1.0/{lyr}/{z:[0-9]+}/{x:[0-9]+}/{y:[0-9]+}", t.ServeTileRequest).Methods("GET")
	t.Router.HandleFunc("/tms/1.0/{lyr}/{z:[0-9]+}/{x:[0-9]+}/{y:[0-9]+}.png", t.ServeTileRequest).Methods("GET")

	return &t
}

// AddMapnikLayer adds mapnik layer to server.
func (self *TileServerPostgresMux) AddMapnikLayer(layerName string, stylesheet string) error {
	Ligneous.Info("Adding tilelayer: ", layerName, " ", stylesheet)

	// check if same layerName exists
	for k := range self.lmp.layerChans {
		if k == layerName {
			Ligneous.Error("Tile layer already exists: ", k)
			return fmt.Errorf("Tile layer already exists: %v", k)
		}
	}

	// Validate source
	if !isValidTileSource(stylesheet) {
		Ligneous.Error("Tile layer source is not valid: ", stylesheet)
		return fmt.Errorf("Tile layer source is not valid: %v", stylesheet)
	}

	// add tile layer
	self.m.AddLayerMetadata(layerName, stylesheet)
	self.lmp.AddRenderer(layerName, stylesheet)
	return nil
}

// GetTileLayer gets metadata for tilelayer.
func (self *TileServerPostgresMux) GetTileLayer(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	vars := mux.Vars(r)
	lyr := vars["lyr"]
	metadata, err := self.m.MetaDataHandler(lyr)
	if nil != err {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		Ligneous.Critical(fmt.Sprintf("%v %v %v [500]", r.RemoteAddr, r.URL.Path, time.Since(start)))
		return
	}
	if _, ok := self.lmp.layerChans[lyr]; !ok {
		http.Error(w, "layer not found", http.StatusNotFound)
		Ligneous.Error(fmt.Sprintf("%v %v %v [404]", r.RemoteAddr, r.URL.Path, time.Since(start)))
		return
	}

	SendJsonResponseFromInterface(w, r, metadata)
}

// NewTileLayer creates new tile layer.
func (self *TileServerPostgresMux) NewTileLayer(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	body, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	if nil != err {
		Ligneous.Critical(fmt.Sprintf("%v %v %v [500]", r.RemoteAddr, r.URL.Path, time.Since(start)))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	api_request := new(ApiRequest)
	err = json.Unmarshal(body, &api_request)
	if nil != err {
		Ligneous.Error(fmt.Sprintf("%v %v %v [400]", r.RemoteAddr, r.URL.Path, time.Since(start)))
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = self.AddMapnikLayer(api_request.Data.TileLayerName, api_request.Data.TileLayerSource)
	if nil != err {
		Ligneous.Error(fmt.Sprintf("%v %v %v [409]", r.RemoteAddr, r.URL.Path, time.Since(start)))
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}

	Ligneous.Info(fmt.Sprintf("%v", api_request))
	Ligneous.Info(fmt.Sprintf("%v %v %v [200]", r.RemoteAddr, r.URL.Path, time.Since(start)))

	SendJsonResponseFromString(`{"status": "ok"}`, w, r)
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

	Ligneous.Info(fmt.Sprintf("%v %v %v [200]", r.RemoteAddr, r.URL.Path, time.Since(start)))
}

// TMSTileMaps lists available TileMaps
func (self *TileServerPostgresMux) TMSTileMaps(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	var layers []string
	for k := range self.lmp.layerChans {
		layers = append(layers, k)
	}
	TMSTileMaps(start, layers, w, r)
}

// TMSTileMap shows list of TileSets for layer
func (self *TileServerPostgresMux) TMSTileMap(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	vars := mux.Vars(r)
	lyr := vars["lyr"]
	metadata, err := self.m.MetaDataHandler(lyr)
	if nil != err {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		Ligneous.Info(fmt.Sprintf("%v %v %v [500]", r.RemoteAddr, r.URL.Path, time.Since(start)))
		return
	}
	if _, ok := self.lmp.layerChans[lyr]; !ok {
		http.Error(w, "layer not found", http.StatusNotFound)
		Ligneous.Info(fmt.Sprintf("%v %v %v [404]", r.RemoteAddr, r.URL.Path, time.Since(start)))
	} else {
		TMSTileMap(start, lyr, metadata["source"], w, r)
	}
}

// ServerProfileHandler returns basic server stats.
func (self *TileServerPostgresMux) ServerProfileHandler(w http.ResponseWriter, r *http.Request) {
	ServerProfileHandler(self.startTime, w, r)
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
	status := SendJsonResponseFromInterface(w, r, response)
	Ligneous.Info(fmt.Sprintf("%v %v %v [200]", r.RemoteAddr, r.URL.Path, time.Since(start), status))
}
