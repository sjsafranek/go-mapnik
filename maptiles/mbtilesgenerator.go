package maptiles

import (
	// "crypto/md5"
	"database/sql"
	"log"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

var logger *log.Logger = log.New(os.Stdout, "[TILE] ", log.LUTC|log.Ldate|log.Ltime|log.Lshortfile|log.Lmicroseconds)

// MBTiles 1.2-compatible Tile Db with multi-layer support.
// Was named Mbtiles before, hence the use of *m in methods.
type TileDb struct {
	db          *sql.DB
	requestChan chan TileFetchRequest
	insertChan  chan TileFetchResult
	layerIds    map[string]int
	qc          chan bool
}

func NewTileDb(path string) *TileDb {
	m := TileDb{}
	var err error
	m.db, err = sql.Open("sqlite3", path)
	if err != nil {
		logger.Println("Error opening db", err.Error())
		return nil
	}
	queries := []string{
		"PRAGMA journal_mode = OFF",
		"CREATE TABLE IF NOT EXISTS layers(layer_name TEXT PRIMARY KEY NOT NULL)",
		"CREATE TABLE IF NOT EXISTS metadata (name TEXT PRIMARY KEY NOT NULL, value TEXT NOT NULL)",
		"CREATE TABLE IF NOT EXISTS tiles (layer_id INTEGER, zoom_level INTEGER, tile_column INTEGER, tile_row INTEGER, tile_data blob, PRIMARY KEY (layer_id, zoom_level, tile_column, tile_row))",
		"REPLACE INTO metadata VALUES('name', 'go-mapnik cache file')",
		"REPLACE INTO metadata VALUES('type', 'overlay')", //baselayer
		"REPLACE INTO metadata VALUES('version', '1')",
		"REPLACE INTO metadata VALUES('description', 'Compatible with MBTiles spec 1.2. However, this file may contain multiple overlay layers, but only the layer called default is exported as MBtiles')",
		"REPLACE INTO metadata VALUES('format', 'png')",
		"REPLACE INTO metadata VALUES('bounds', '-180.0,-85,180,85')",
		"REPLACE INTO metadata VALUES('attribution', 'sjsafranek')",
		"INSERT OR IGNORE INTO layers(layer_name) VALUES('default')",
	}

	for _, query := range queries {
		_, err = m.db.Exec(query)
		if err != nil {
			logger.Println("Error setting up db", err.Error())
			logger.Println(query, "\n");
			return nil
		}
	}

	m.readLayers()

	m.insertChan = make(chan TileFetchResult)
	m.requestChan = make(chan TileFetchRequest)
	go m.Run()
	return &m
}

func (self *TileDb) readLayers() {
	self.layerIds = make(map[string]int)
	rows, err := self.db.Query("SELECT rowid, layer_name FROM layers")
	if err != nil {
		logger.Fatal("Error fetching layer definitions", err.Error())
	}
	var s string
	var i int
	for rows.Next() {
		if err := rows.Scan(&i, &s); err != nil {
			logger.Fatal(err)
		}
		self.layerIds[s] = i
	}
	if err := rows.Err(); err != nil {
		logger.Fatal(err)
	}
}

func (self *TileDb) ensureLayer(layer string) {
	if _, ok := self.layerIds[layer]; !ok {
		if _, err := self.db.Exec("INSERT OR IGNORE INTO layers(layer_name) VALUES(?)", layer); err != nil {
			logger.Println(err)
		}
		self.readLayers()
	}
}

func (self *TileDb) Close() {
	close(self.insertChan)
	close(self.requestChan)
	if self.qc != nil {
		<-self.qc // block until channel qc is closed (meaning Run() is finished)
	}
	if err := self.db.Close(); err != nil {
		logger.Print(err)
	}

}

func (self TileDb) InsertQueue() chan<- TileFetchResult {
	return self.insertChan
}

func (self TileDb) RequestQueue() chan<- TileFetchRequest {
	return self.requestChan
}

// Best executed in a dedicated go routine.
func (self *TileDb) Run() {
	self.qc = make(chan bool)
	for {
		select {
		case r := <-self.requestChan:
			self.fetch(r)
		case i := <-self.insertChan:
			self.insert(i)
		}
	}
	self.qc <- true
}

func (self *TileDb) insert(i TileFetchResult) {
	i.Coord.setTMS(true)
	x, y, zoom, l := i.Coord.X, i.Coord.Y, i.Coord.Zoom, i.Coord.Layer
	if l == "" {
		l = "default"
	}
	// h := md5.New()
	// _, err := h.Write(i.BlobPNG)
	// if err != nil {
	// 	logger.Println(err)
	// 	return
	// }
	// s := fmt.Sprintf("%x", h.Sum(nil))
	// row := self.db.QueryRow("SELECT 1 FROM tile_blobs WHERE checksum=?", s)
	queryString := "SELECT tile_data FROM tiles WHERE layer_id=? AND zoom_level=? AND tile_column=? AND tile_row=?"
	row := self.db.QueryRow(queryString, self.layerIds[l], zoom, x, y)
	var dummy uint64
	err := row.Scan(&dummy)
	switch {
	case err == sql.ErrNoRows:
		// queryString = "REPLACE INTO tiles(tile_data) VALUES(?) WHERE layer_id=? AND zoom_level=? AND tile_column=? AND tile_row=?"
		queryString = "UPDATE tiles SET tile_data=? WHERE layer_id=? AND zoom_level=? AND tile_column=? AND tile_row=?"
		if _, err = self.db.Exec(queryString, i.BlobPNG, self.layerIds[l], zoom, x, y); err != nil {
			logger.Println("error during insert", err)
			return
		}
	case err != nil:
		logger.Println("error during test", err)
		return
	default:
		logger.Println("Reusing blob", self.layerIds[l], zoom, x, y)
	}
	self.ensureLayer(l)
	// queryString = "REPLACE INTO tiles VALUES(?, ?, ?, ?, ?)"
	queryString = "REPLACE INTO tiles VALUES(?, ?, ?, ?, ?)"
	if _, err = self.db.Exec(queryString, self.layerIds[l], zoom, x, y, i.BlobPNG); err != nil {
		logger.Println(err)
	}
}

func (self *TileDb) fetch(r TileFetchRequest) {
	r.Coord.setTMS(true)
	zoom, x, y, l := r.Coord.Zoom, r.Coord.X, r.Coord.Y, r.Coord.Layer
	if l == "" {
		l = "default"
	}
	result := TileFetchResult{r.Coord, nil}
	queryString := `
		SELECT tile_data 
		FROM tiles 
		WHERE zoom_level=? 
			AND tile_column=? 
			AND tile_row=?
			AND layer_id=?
		`
	var blob []byte
	row := self.db.QueryRow(queryString, zoom, x, y, self.layerIds[l])
	err := row.Scan(&blob)
	switch {
	case err == sql.ErrNoRows:
		result.BlobPNG = nil
	case err != nil:
		logger.Println(err)
	default:
		result.BlobPNG = blob
	}
	r.OutChan <- result
}

// gets tile data
func (self *TileDb) MetaDataHandler() map[string]string {
	rows, _ := self.db.Query("SELECT * FROM metadata")
	metadata :=  make(map[string]string)
	for rows.Next() {
		var name string
		var value string
		rows.Scan(&name, &value)
		metadata[name] = value
	}
	return metadata
}
