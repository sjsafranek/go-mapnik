package maptiles

import (
	"database/sql"
	_ "github.com/lib/pq"
)

// MBTiles 1.2-compatible Tile Db with multi-layer support.
// Was named Mbtiles before, hence the use of *m in methods.
type TileDbPostgresql struct {
	db          *sql.DB
	requestChan chan TileFetchRequest
	insertChan  chan TileFetchResult
	layerIds    map[string]int
	qc          chan bool
}

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
		"CREATE TABLE IF NOT EXISTS metadata (name TEXT PRIMARY KEY NOT NULL, value TEXT NOT NULL, layer_name TEXT NOT NULL);",
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

		// "INSERT INTO metadata(name,value,layer_name) VALUES('name', 'go-mapnik cache file', 'default')",
		// "INSERT INTO metadata(name,value,layer_name) VALUES('type', 'overlay', 'default')", //baselayer
		// "INSERT INTO metadata(name,value,layer_name) VALUES('version', '1', 'default')",
		// "INSERT INTO metadata(name,value,layer_name) VALUES('description', 'Compatible with MBTiles spec 1.2. However, this file may contain multiple overlay layers, but only the layer called default is exported as MBtiles', 'default')",
		// "INSERT INTO metadata(name,value,layer_name) VALUES('format', 'png', 'default')",
		// "INSERT INTO metadata(name,value,layer_name) VALUES('bounds', '-180.0,-85,180,85', 'default')",
		// "INSERT INTO metadata(name,value,layer_name) VALUES('attribution', 'sjsafranek', 'default')",
		// "INSERT INTO layers(layer_name) SELECT 'default' WHERE NOT EXISTS (SELECT layer_name FROM layers WHERE layer_name='default');",
		// "INSERT INTO layers(layer_name) VALUES ('default')",
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

func (self *TileDbPostgresql) ensureLayer(layer string) {
	if _, ok := self.layerIds[layer]; !ok {
		// queryString := "INSERT OR IGNORE INTO layers(layer_name) VALUES($1)"
		queryString := "INSERT INTO layers(layer_name) VALUES($1)"
		if _, err := self.db.Exec(queryString, layer); err != nil {
			Ligneous.Debug(err)
		}
		self.readLayers()
	}
}

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

func (self TileDbPostgresql) InsertQueue() chan<- TileFetchResult {
	return self.insertChan
}

func (self TileDbPostgresql) RequestQueue() chan<- TileFetchRequest {
	return self.requestChan
}

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
		Ligneous.Trace("Insert blob", self.layerIds[l], zoom, x, y)
	case err != nil:
		Ligneous.Error("error during test", err)
		return
	default:
		Ligneous.Trace("Insert blob", self.layerIds[l], zoom, x, y)
	}
	self.ensureLayer(l)
	// queryString = "REPLACE INTO tiles VALUES($1, $2, $3, $4, $5)"
	queryString = "INSERT INTO tiles VALUES($1, $2, $3, $4, $5)"
	if _, err = self.db.Exec(queryString, self.layerIds[l], zoom, x, y, i.BlobPNG); err != nil {
		Ligneous.Error(err)
	}
}

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
		Ligneous.Trace("Reusing blob ", self.layerIds[l], zoom, x, y)
	}
	r.OutChan <- result
}

// gets tile data
func (self *TileDbPostgresql) MetaDataHandler() map[string]string {
	rows, _ := self.db.Query("SELECT * FROM metadata")
	metadata := make(map[string]string)
	for rows.Next() {
		var name string
		var value string
		rows.Scan(&name, &value)
		metadata[name] = value
	}
	return metadata
}
