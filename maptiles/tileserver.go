package maptiles

import (
	"log"
	"net/http"
	"regexp"
	"strconv"

	"encoding/json"
	"database/sql"
	"fmt"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

// TODO serve list of registered layers per HTTP (preferably leafletjs-compatible js-array)

// Handles HTTP requests for map tiles, caching any produced tiles
// in an MBtiles 1.2 compatible sqlite db.
type TileServer struct {
	m         *TileDb
	lmp       *LayerMultiplex
	TmsSchema bool
}

func NewTileServer(cacheFile string) *TileServer {
	t := TileServer{}
	t.lmp = NewLayerMultiplex()
	t.m = NewTileDb(cacheFile)

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

	// if string.Contains(r.URL.Path, "metadata") {
	// 	t.MetaDataHandler(w, r)
	// 	return;
	// }

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



// func (t *TileServer) MetaDataHandler(w http.ResponseWriter, r *http.Request) {
// 	// Set headers
// 	w.Header().Set("Content-Type", "application/json")
// 	w.Header().Set("Access-Control-Allow-Origin", "*")

// 	// Get params
// 	vars := mux.Vars(r)
// 	dbname := vars["db"]

// 	// check for file
// 	if _, err := os.Stat(dbname+".mbtiles"); os.IsNotExist(err) {
// 		fmt.Println("File not found [" + dbname + ".mbtiles]")
// 		http.Error(w, err.Error(), http.StatusNotFound)
// 		return
// 	}

// 	// Open database
// 	db, _ := sql.Open("sqlite3", "./"+dbname+".mbtiles")
// 	rows, _ := db.Query("SELECT * FROM metadata")

// 	metadata :=  make(map[string]string)

// 	for rows.Next() {

// 		var name string
// 		var value string
// 		rows.Scan(&name, &value)
		
// 		metadata[name] = value
// 	}

// 	db.Close()

// 	response_wrapper := make(map[string]interface{})
// 	response_wrapper["status"] = "success"
// 	response_wrapper["data"] = metadata

// 	js, err := json.Marshal(response_wrapper)
// 	if err != nil {
// 		http.Error(w, err.Error(), http.StatusInternalServerError)
// 		return
// 	}

// 	w.Write(js)

// }
