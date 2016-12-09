package main

import (
	"net/http"
	"flag"
	"log"
	"os"

	"./maptiles"
)

var (
	config map[string]string
	engine string
	port string
	db_cache string
)

var logger *log.Logger = log.New(os.Stdout, "[GoMapnikTiles] ", log.LUTC|log.Ldate|log.Ltime|log.Lshortfile|log.Lmicroseconds)

// Serve a single stylesheet via HTTP. Open view_tileserver.html in your browser
// to see the results.
// The created tiles are cached in an sqlite database (MBTiles 1.2 conform) so
// successive access a tile is much faster.
func TileserverWithCaching(engine string, layer_config map[string]string) {
	if engine == "postgres" {
		t := maptiles.NewTileServerPostgres(db_cache)
		for i := range layer_config {
			t.AddMapnikLayer(i, layer_config[i])
		}
		logger.Println("Connecting to postgres databas:")
		logger.Println("***", db_cache)
		logger.Printf("Magic happens on port %s...", port)
		logger.Fatal(http.ListenAndServe("0.0.0.0:"+port, t))
	} else {
		t := maptiles.NewTileServerSqlite(db_cache)
		for i := range layer_config {
			t.AddMapnikLayer(i, layer_config[i])
		}
		logger.Println("Connecting to sqlite3 database:")
		logger.Println("***", db_cache)
		logger.Printf("Magic happens on port %s...", port)
		logger.Fatal(http.ListenAndServe("0.0.0.0:"+port, t))
	}

}

func init() {
	// TODO: add config file
	flag.StringVar(&port, "p", "8080", "server port")
	flag.StringVar(&engine, "e", "sqlite", "database engine [sqlite or postgres]")
	flag.StringVar(&db_cache, "d", "gomapnikcache.mbtiles", "tile cache database")
	flag.Parse()
	if engine != "sqlite" {
		if engine != "postgres" {
			logger.Fatal("Unsupported database engines")
		}
	}
}

// Before uncommenting the GenerateOSMTiles call make sure you have
// the neccessary OSM sources. Consult OSM wiki for details.
func main() {
	config = make(map[string]string)
	config["default"] = "sampledata/world/stylesheet.xml"
	config["sample"] = "sampledata/world/stylesheet.xml"
	config["population"] = "sampledata/world_population/population.xml"
	//config["osm"] = "https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png"
	config["osm"] = "https://a.tile.openstreetmap.org/{z}/{x}/{y}.png"
	TileserverWithCaching(engine, config)
}


/*

// create user for linux
stefan@stefan:~$ adduser mapnik
*** password 'dev'

// create new postgres user and database table 
stefan@stefan:~$ sudo -i -u postgres
postgres@stefan:~$ psql
postgres=# CREATE USER mapnik WITH PASSWORD 'dev';
postgres=# CREATE DATABASE mbtiles;
postgres=# GRANT ALL PRIVILEGES ON DATABASE mbtiles TO mapnik;

// check
stefan@stefan:~$ sudo -i -u mapnik
mapnik@stefan:~$ psql -d mbtiles -U mapnik -W dev

// Run with Postgresql
stefan@stefan:~$ go run TileServer.go -e postgres -d postgres://mapnik:dev@localhost/mbtiles

// Run with Sqlite3
stefan@stefan:~$ go run TileServer.go -e sqlite -d gomapnikcache.mbtiles


su - mapnik
sudo -i -u mapnik


*/

