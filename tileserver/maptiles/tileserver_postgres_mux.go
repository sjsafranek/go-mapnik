package maptiles

import (
	"fmt"
	"net/http"
	"runtime"
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
	var root = `<?xml version="1.0" encoding="utf-8" ?>
				 <Services>
				 	<TileMapService title="GoMapnik Tile Map Service" version="1.0" href="http:127.0.0.1/tms/1.0"/>
				 </Services>`
	fmt.Fprintf(w, root)
	Ligneous.Info(fmt.Sprintf("%v %v %v ", r.RemoteAddr, r.URL.Path, time.Since(start)))
}

// TMSTileMaps lists available TileMaps
func (self *TileServerPostgresMux) TMSTileMaps(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	var TileMaps = ``
	for k := range self.lmp.layerChans {
		TileMaps += `<TileMap title="` + k + `" srs="EPSG:4326" href="http:127.0.0.1:8080` + r.URL.Path + `/` + k + `"></TileMap>`
	}
	var tree = `<?xml version="1.0" encoding="utf-8" ?>
				 <TileMapService version="1.0" services="http:127.0.0.1:8080` + r.URL.Path + `">
				 	<Abstract></Abstract>
					<TileMaps>
						` + TileMaps + `
					</TileMaps>
				 </TileMapService>`
	fmt.Fprintf(w, tree)
	Ligneous.Info(fmt.Sprintf("%v %v %v ", r.RemoteAddr, r.URL.Path, time.Since(start)))
}

// TMSTileMap shows list of TileSets for layer
func (self *TileServerPostgresMux) TMSTileMap(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	vars := mux.Vars(r)
	lyr := vars["lyr"]
	// metadata := self.m.MetaDataHandler(lyr)
	var TileSets = ``
	for i := 0; i < 21; i++ {
		TileSets += `<TileSet
						href="` + fmt.Sprintf("http:127.0.0.1:8080%v/%v", r.URL.Path, i) + `"
						units-per-pixel="` + fmt.Sprintf("%v", unitsPerPixel(i)) + `"
						order="` + fmt.Sprintf("%v", i) + `">
					</TileSet>`
	}
	var tree = `<?xml version="1.0" encoding="utf-8" ?>
				 <TileMap version="1.0" services="http:127.0.0.1:8080` + r.URL.Path + `">
				 	<Title>` + lyr + `</Title>
					<Abstract></Abstract>
					<SRS>EPSG:4326</SRS>
					<BoundingBox minx="-180" miny="-90" maxx="180" max="90"></BoundingBox>
					<Origin x="-180" y="-90"></Origin>
					<TileFormat width="256" height="256" mime-type="image/png" extension="png"></TileFormat>
					<TileSets profile="global-geodetic">
						` + TileSets + `
					</TileSets>
				 </TileMap>`
	fmt.Fprintf(w, tree)
	Ligneous.Info(fmt.Sprintf("%v %v %v ", r.RemoteAddr, r.URL.Path, time.Since(start)))
}

// TMSErrorTile returns error response
func (self *TileServerPostgresMux) TMSErrorTile(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	message := "Expecting /{layer}/{z}/{x}/{y}.png"
	http.Error(w, message, http.StatusBadRequest)
	Ligneous.Info(fmt.Sprintf("%v %v %v ", r.RemoteAddr, r.URL.Path, time.Since(start)))
}

// MetadataHandler for tile server.
func (self *TileServerPostgresMux) MetadataHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	vars := mux.Vars(r)
	lyr := vars["lyr"]
	metadata := self.m.MetaDataHandler(lyr)
	response := make(map[string]interface{})
	response["status"] = "ok"
	response["data"] = metadata
	SendJsonResponseFromInterface(w, r, response)
	Ligneous.Info(fmt.Sprintf("%v %v %v ", r.RemoteAddr, r.URL.Path, time.Since(start)))
}

// PingHandler provides an api route for server health check
func (self *TileServerPostgresMux) PingHandler(w http.ResponseWriter, r *http.Request) {
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
func (self *TileServerPostgresMux) ServerProfileHandler(w http.ResponseWriter, r *http.Request) {
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
