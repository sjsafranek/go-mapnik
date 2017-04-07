package maptiles

import (
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
)

// TileDbPostgresql struct for PostgreSQL MBTile database.
// MBTiles 1.2-compatible Tile Db with multi-layer support.
// Was named Mbtiles before, hence the use of *m in methods.
type TileDbPostgresql struct {
	db          *sql.DB
	requestChan chan TileFetchRequest
	insertChan  chan TileFetchResult
	layerIds    map[string]int
	qc          chan bool
}

// NewTileDbPostgresql creates TileDbPostgresql struct.
// Creates database tables and initializes tile request channels.
func NewTileDbPostgresql(path string) *TileDbPostgresql {
	m := TileDbPostgresql{}
	var err error
	m.db, err = sql.Open("postgres", path)
	if err != nil {
		Ligneous.Error(err)
		return nil
	}
	queries := []string{
		// Table: layers
		"CREATE TABLE IF NOT EXISTS layers (layer_name TEXT PRIMARY KEY NOT NULL, rowid SERIAL);",
		"COMMENT ON TABLE layers IS 'Names of tile layers';",
		"COMMENT ON COLUMN layers.layer_name IS 'Tile layer name';",
		"COMMENT ON COLUMN layers.rowid IS 'Tile layer index';",

		// Table: metadata
		"CREATE TABLE IF NOT EXISTS metadata (name TEXT NOT NULL, value TEXT NOT NULL, layer_name TEXT NOT NULL);",
		"COMMENT ON TABLE metadata IS 'Metadata for tile server layers';",
		"COMMENT ON COLUMN metadata.name IS 'metadata map name';",
		"COMMENT ON COLUMN metadata.value IS 'metadata map value';",
		"COMMENT ON COLUMN metadata.layer_name IS 'metadata map layer_name';",

		// Table: tiles
		"CREATE TABLE IF NOT EXISTS tiles (layer_id INTEGER, zoom_level INTEGER, tile_column INTEGER, tile_row INTEGER, tile_data BYTEA);",
		"COMMENT ON TABLE tiles IS 'Cached png map tiles';",
		"COMMENT ON COLUMN tiles.layer_id IS 'layer id for table join';",
		"COMMENT ON COLUMN tiles.zoom_level IS 'png tile zoom';",
		"COMMENT ON COLUMN tiles.tile_column IS 'png tile column';",
		"COMMENT ON COLUMN tiles.tile_row IS 'png tile row';",
		"COMMENT ON COLUMN tiles.tile_data IS 'png tile data';",
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
func (self *TileDbPostgresql) readLayers() {
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
func (self *TileDbPostgresql) ensureLayer(layer string) {
	if _, ok := self.layerIds[layer]; !ok {
		queryString := "INSERT INTO layers(layer_name) VALUES($1)"
		if _, err := self.db.Exec(queryString, layer); err != nil {
			Ligneous.Debug(err)
		}
		self.readLayers()
	}
}

// Close tile request channels.
func (self *TileDbPostgresql) Close() {
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
func (self TileDbPostgresql) InsertQueue() chan<- TileFetchResult {
	return self.insertChan
}

// RequestQueue gets tile request channel.
func (self TileDbPostgresql) RequestQueue() chan<- TileFetchRequest {
	return self.requestChan
}

// Run runs tile generation.
// Best executed in a dedicated go routine.
func (self *TileDbPostgresql) Run() {
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
func (self *TileDbPostgresql) insert(i TileFetchResult) {
	i.Coord.setTMS(true)
	x, y, zoom, l := i.Coord.X, i.Coord.Y, i.Coord.Zoom, i.Coord.Layer
	queryString := "SELECT tile_data FROM tiles WHERE layer_id=$1 AND zoom_level=$2 AND tile_column=$3 AND tile_row=$4"
	row := self.db.QueryRow(queryString, self.layerIds[l], zoom, x, y)
	var dummy uint64
	err := row.Scan(&dummy)
	switch {
	case err == sql.ErrNoRows:
		queryString = "UPDATE tiles SET tile_data=$1 WHERE layer_id=$2 AND zoom_level=$3 AND tile_column=$4 AND tile_row=$5"
		if _, err = self.db.Exec(queryString, i.BlobPNG, self.layerIds[l], zoom, x, y); err != nil {
			Ligneous.Error("error during insert", err)
			return
		}
		Ligneous.Trace(fmt.Sprintf("INSERT BLOB %v %v %v %v", l, zoom, x, y))
	case err != nil:
		Ligneous.Error("error during test", err)
		return
	default:
		Ligneous.Trace(fmt.Sprintf("INSERT BLOB %v %v %v %v", l, zoom, x, y))
	}
	self.ensureLayer(l)
	queryString = "INSERT INTO tiles VALUES($1, $2, $3, $4, $5)"
	if _, err = self.db.Exec(queryString, self.layerIds[l], zoom, x, y, i.BlobPNG); err != nil {
		Ligneous.Error(err)
	}
}

// fetch gets cached tile from database.
func (self *TileDbPostgresql) fetch(r TileFetchRequest) {
	r.Coord.setTMS(true)
	zoom, x, y, l := r.Coord.Zoom, r.Coord.X, r.Coord.Y, r.Coord.Layer
	result := TileFetchResult{r.Coord, nil}
	queryString := `
		SELECT tile_data
		FROM tiles
		WHERE zoom_level=$1
			AND tile_column=$2
			AND tile_row=$3
			AND layer_id=$4
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
func (self *TileDbPostgresql) AddLayerMetadata(lyr string, stylesheet string) {

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
func (self *TileDbPostgresql) rowExists(query string) bool {
	var exists bool
	err := self.db.QueryRow(query).Scan(&exists)
	if nil != err {
		Ligneous.Error(err)
	}
	return exists
}

// MetaDataHandler gets metadata from database.
func (self *TileDbPostgresql) MetaDataHandler(lyr string) (map[string]string, error) {
	metadata := make(map[string]string)
	rows, err := self.db.Query("SELECT name, value FROM metadata WHERE layer_name=$1", lyr)
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
func (self *TileDbPostgresql) GetTileLayers() (map[string]map[string]string, error) {
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
