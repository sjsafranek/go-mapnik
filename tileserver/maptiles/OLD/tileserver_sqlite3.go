package maptiles

import (
	"fmt"
	"net/http"
	"runtime"
	"time"
)

// TileServerSqlite Sqlite3 tile server.
// Handles HTTP requests for map tiles, caching any produced tiles
// in an MBtiles 1.2 compatible sqlite db.
type TileServerSqlite struct {
	engine    string
	m         *TileDbSqlite3
	lmp       *LayerMultiplex
	TmsSchema bool
	startTime time.Time
}

// NewTileServerSqlite creates TileServerSqlite object.
func NewTileServerSqlite(cacheFile string) *TileServerSqlite {
	t := TileServerSqlite{}
	t.lmp = NewLayerMultiplex()
	t.m = NewTileDbSqlite(cacheFile)
	t.startTime = time.Now()
	return &t
}

// AddMapnikLayer adds mapnik layer to server.
func (self *TileServerSqlite) AddMapnikLayer(layerName string, stylesheet string) {
	self.lmp.AddRenderer(layerName, stylesheet)
}

// ServeTileRequest serves tile request.
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

	w.Header().Set("Content-Type", "image/png")
	w.WriteHeader(http.StatusOK)

	_, err := w.Write(result.BlobPNG)
	if err != nil {
		Ligneous.Error(err)
	}
	if needsInsert {
		self.m.InsertQueue() <- result // insert newly rendered tile into cache db
	}
}

// ServeHTTP http server.
func (self *TileServerSqlite) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	start := time.Now()

	if "/" == r.URL.Path {
		Ligneous.Info(fmt.Sprintf("%v %v %v ", r.RemoteAddr, r.URL.Path, time.Since(start)))
		self.IndexHandler(w, r)
		return
	} else if "/ping" == r.URL.Path {
		Ligneous.Info(fmt.Sprintf("%v %v %v ", r.RemoteAddr, r.URL.Path, time.Since(start)))
		self.PingHandler(w, r)
		return
	} else if "/server" == r.URL.Path {
		Ligneous.Info(fmt.Sprintf("%v %v %v ", r.RemoteAddr, r.URL.Path, time.Since(start)))
		self.ServerHandler(w, r)
		return
	} else if "/metadata" == r.URL.Path {
		Ligneous.Info(fmt.Sprintf("%v %v %v ", r.RemoteAddr, r.URL.Path, time.Since(start)))
		self.MetadataHandler(w, r)
		return
	} else if "/tilelayers" == r.URL.Path {
		Ligneous.Info(fmt.Sprintf("%v %v %v ", r.RemoteAddr, r.URL.Path, time.Since(start)))
		self.TileLayersHandler(w, r)
		return
	}

	layer, xyz, err := ParseTileUrl(r.URL.Path)
	if nil != err {
		Ligneous.Info(fmt.Sprintf("%v %v %v ", r.RemoteAddr, r.URL.Path, time.Since(start)))
		self.RequestErrorHandler(w, r)
		return
	}

	self.ServeTileRequest(w, r, TileCoord{xyz[0], xyz[1], xyz[2], self.TmsSchema, layer})

	Ligneous.Info(fmt.Sprintf("%v %v %v ", r.RemoteAddr, r.URL.Path, time.Since(start)))
}

// RequestErrorHandler handles error response.
func (self *TileServerSqlite) RequestErrorHandler(w http.ResponseWriter, r *http.Request) {
	response := make(map[string]interface{})
	response["status"] = "error"
	result := make(map[string]interface{})
	result["message"] = "Expecting /{datasource}/{z}/{x}/{y}.png"
	response["data"] = result
	SendJsonResponseFromInterface(w, r, response)
}

// IndexHandler for server.
func (self *TileServerSqlite) IndexHandler(w http.ResponseWriter, r *http.Request) {
	response := make(map[string]interface{})
	response["status"] = "ok"
	result := make(map[string]interface{})
	result["message"] = "Hello there ladies and gentlemen!"
	response["data"] = result
	SendJsonResponseFromInterface(w, r, response)
}

// MetadataHandler for tile server.
func (self *TileServerSqlite) MetadataHandler(w http.ResponseWriter, r *http.Request) {
	// todo: include layer
	metadata := self.m.MetaDataHandler()
	response := make(map[string]interface{})
	response["status"] = "ok"
	response["data"] = metadata
	SendJsonResponseFromInterface(w, r, response)
}

// PingHandler provides an api route for server health check.
func (self *TileServerSqlite) PingHandler(w http.ResponseWriter, r *http.Request) {
	response := make(map[string]interface{})
	response["status"] = "ok"
	result := make(map[string]interface{})
	result["result"] = "Pong"
	response["data"] = result
	SendJsonResponseFromInterface(w, r, response)
}

// ServerHandler returns basic server stats.
func (self *TileServerSqlite) ServerHandler(w http.ResponseWriter, r *http.Request) {
	var data map[string]interface{}
	data = make(map[string]interface{})
	data["registered"] = self.startTime.UTC()
	data["uptime"] = time.Since(self.startTime).Seconds()
	data["num_cores"] = runtime.NumCPU()
	response := make(map[string]interface{})
	response["status"] = "ok"
	response["data"] = data
	//response["free_mem"] = runtime.MemStats()
	SendJsonResponseFromInterface(w, r, response)
}

// TileLayersHandler returns list of tiles.
func (self *TileServerSqlite) TileLayersHandler(w http.ResponseWriter, r *http.Request) {
	var keys []string
	for k := range self.lmp.layerChans {
		keys = append(keys, k)
	}
	var response map[string]interface{}
	response = make(map[string]interface{})
	response["status"] = "ok"
	response["data"] = keys
	SendJsonResponseFromInterface(w, r, response)
}
