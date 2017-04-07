package maptiles

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
)

// TileDbSqlite3 struct for SQLite3 MBTile database.
// MBTiles 1.2-compatible Tile Db with multi-layer support.
// Was named Mbtiles before, hence the use of *m in methods.
type TileDbSqlite3 struct {
	db          *sql.DB
	requestChan chan TileFetchRequest
	insertChan  chan TileFetchResult
	layerIds    map[string]int
	qc          chan bool
}

// NewTileDbSqlite creates TileDbSqlite3 struct.
// Creates database tables and initializes tile request channels.
func NewTileDbSqlite(path string) *TileDbSqlite3 {
	m := TileDbSqlite3{}
	var err error
	m.db, err = sql.Open("sqlite3", path)
	if err != nil {
		Ligneous.Error("Error opening db", err.Error())
		return nil
	}
	queries := []string{
		"PRAGMA journal_mode = OFF",
		"CREATE TABLE IF NOT EXISTS layers(layer_name TEXT PRIMARY KEY NOT NULL)",
		"CREATE TABLE IF NOT EXISTS metadata (name TEXT NOT NULL, value TEXT NOT NULL, layer_name TEXT NOT NULL)",
		"CREATE TABLE IF NOT EXISTS tiles (layer_id INTEGER, zoom_level INTEGER, tile_column INTEGER, tile_row INTEGER, tile_data blob, PRIMARY KEY (layer_id, zoom_level, tile_column, tile_row))",
	}

	for _, query := range queries {
		_, err = m.db.Exec(query)
		if err != nil {
			Ligneous.Error("Error setting up db", err.Error())
			Ligneous.Debug(query, "\n")
			return nil
		}
	}

	m.readLayers()

	m.insertChan = make(chan TileFetchResult)
	m.requestChan = make(chan TileFetchRequest)
	go m.Run()
	return &m
}

// readLayers reads through tile layers table and sets up
// lookup table for layer names and indexes.
func (self *TileDbSqlite3) readLayers() {
	self.layerIds = make(map[string]int)
	rows, err := self.db.Query("SELECT rowid, layer_name FROM layers")
	if err != nil {
		Ligneous.Error("Error fetching layer definitions", err.Error())
	}
	var s string
	var i int
	for rows.Next() {
		if err := rows.Scan(&i, &s); err != nil {
			Ligneous.Error(err)
		}
		self.layerIds[s] = i
	}
	if err := rows.Err(); err != nil {
		Ligneous.Error(err)
	}
}

// ensureLayer checks if tile layer is in lookup table.
func (self *TileDbSqlite3) ensureLayer(layer string) {
	if _, ok := self.layerIds[layer]; !ok {
		if _, err := self.db.Exec("INSERT OR IGNORE INTO layers(layer_name) VALUES(?)", layer); err != nil {
			Ligneous.Error(err)
		}
		self.readLayers()
	}
}

// Close tile request channels.
func (self *TileDbSqlite3) Close() {
	close(self.insertChan)
	close(self.requestChan)
	if self.qc != nil {
		<-self.qc // block until channel qc is closed (meaning Run() is finished)
	}
	if err := self.db.Close(); err != nil {
		Ligneous.Error(err)
	}

}

// InsertQueue gets tile insert channel.
func (self TileDbSqlite3) InsertQueue() chan<- TileFetchResult {
	return self.insertChan
}

// RequestQueue gets tile request channel.
func (self TileDbSqlite3) RequestQueue() chan<- TileFetchRequest {
	return self.requestChan
}

// Run runs tile generation.
// Best executed in a dedicated go routine.
func (self *TileDbSqlite3) Run() {
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

// insert tile request into database table.
func (self *TileDbSqlite3) insert(i TileFetchResult) {
	i.Coord.setTMS(true)
	x, y, zoom, l := i.Coord.X, i.Coord.Y, i.Coord.Zoom, i.Coord.Layer
	queryString := "SELECT tile_data FROM tiles WHERE layer_id=? AND zoom_level=? AND tile_column=? AND tile_row=?"
	row := self.db.QueryRow(queryString, self.layerIds[l], zoom, x, y)
	var dummy uint64
	err := row.Scan(&dummy)
	switch {
	case err == sql.ErrNoRows:
		queryString = "UPDATE tiles SET tile_data=? WHERE layer_id=? AND zoom_level=? AND tile_column=? AND tile_row=?"
		if _, err = self.db.Exec(queryString, i.BlobPNG, self.layerIds[l], zoom, x, y); err != nil {
			Ligneous.Error("error during insert", err)
			return
		}
		Ligneous.Trace(fmt.Sprintf("INSERT BLOB %v %v %v %v", l, zoom, x, y))
	case err != nil:
		Ligneous.Error(err)
		return
	default:
		Ligneous.Trace(fmt.Sprintf("INSERT BLOB %v %v %v %v", l, zoom, x, y))
	}
	self.ensureLayer(l)
	queryString = "REPLACE INTO tiles VALUES(?, ?, ?, ?, ?)"
	if _, err = self.db.Exec(queryString, self.layerIds[l], zoom, x, y, i.BlobPNG); err != nil {
		Ligneous.Error(err)
	}
}

// fetch gets cached tile from database.
func (self *TileDbSqlite3) fetch(r TileFetchRequest) {
	r.Coord.setTMS(true)
	zoom, x, y, l := r.Coord.Zoom, r.Coord.X, r.Coord.Y, r.Coord.Layer
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
		Ligneous.Error(err)
	default:
		result.BlobPNG = blob
		Ligneous.Trace(fmt.Sprintf("REUSE BLOB %v %v %v %v", l, zoom, x, y))
	}
	r.OutChan <- result
}

// AddLayerMetadata adds metadata t0 metadata table
func (self *TileDbSqlite3) AddLayerMetadata(lyr string, stylesheet string) {

	check_query := "SELECT EXISTS(SELECT * FROM metadata WHERE name='name' AND layer_name='" + lyr + "')"

	insert_queries := []string{
		"INSERT INTO metadata(name, value, layer_name) VALUES('name', '" + lyr + "', '" + lyr + "')",
		"INSERT INTO metadata(name, value, layer_name) VALUES('source', '" + stylesheet + "', '" + lyr + "')",
		"INSERT INTO metadata(name, value, layer_name) VALUES('type', 'overlay', '" + lyr + "')",
		"INSERT INTO metadata(name,value,layer_name) VALUES('version', '1', '" + lyr + "')",
		"INSERT INTO metadata(name,value,layer_name) VALUES('description', 'Compatible with MBTiles spec 1.2.', '" + lyr + "')",
		"INSERT INTO metadata(name,value,layer_name) VALUES('format', 'png', '" + lyr + "')",
		"INSERT INTO metadata(name,value,layer_name) VALUES('bounds', '-180.0,-85,180,85', '" + lyr + "')",
		"INSERT INTO metadata(name,value,layer_name) VALUES('attribution', 'sjsafranek', '" + lyr + "')",
	}

	if !self.rowExists(check_query) {
		Ligneous.Info("Adding metadata for ", lyr)
		for _, query := range insert_queries {
			_, err := self.db.Exec(query)
			if err != nil {
				Ligneous.Error("Error adding metadata to db", err.Error())
				Ligneous.Debug(query, "\n")
			}
		}
		self.ensureLayer(lyr)
	}

}

// rowExists checks if row exists in table
func (self *TileDbSqlite3) rowExists(query string) bool {
	var exists bool
	err := self.db.QueryRow(query).Scan(&exists)
	if nil != err {
		Ligneous.Error(err)
	}
	return exists
}

// MetaDataHandler gets metadata from database.
func (self *TileDbSqlite3) MetaDataHandler(lyr string) (map[string]string, error) {
	metadata := make(map[string]string)
	rows, err := self.db.Query("SELECT name, value FROM metadata WHERE layer_name=?", lyr)
	if nil != err {
		Ligneous.Error(err)
		return metadata, err
	}
	for rows.Next() {
		var name string
		var value string
		rows.Scan(&name, &value)
		metadata[name] = value
	}
	return metadata, nil
}

// GetTileLayers get metadata for all tilelayers.
func (self *TileDbSqlite3) GetTileLayers() (map[string]map[string]string, error) {
	layers := make(map[string]map[string]string)
	rows, err := self.db.Query("SELECT layer_name FROM layers")
	if nil != err {
		Ligneous.Error(err)
		return layers, err
	}
	for rows.Next() {
		var layer_name string
		rows.Scan(&layer_name)
		metadata, err := self.MetaDataHandler(layer_name)
		if nil != err {
			Ligneous.Error(err)
			return layers, err
		}
		layers[layer_name] = metadata
	}
	return layers, nil
}
