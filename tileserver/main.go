package main

import (
	"fmt"
	"net/http"
	"flag"
	"log"
	"os"
	"io/ioutil"
	"encoding/json"

	"tileserver/maptiles"
)

type Config struct {
	Cache string `json:"cache"`
	Engine string `json:"engine"`
	Layers map[string]string `json:"layers"`
	Port int `json:"port"`
}

var (
	config Config
	//engine string
	//port string
	//db_cache string
	config_file string
)

var logger *log.Logger = log.New(os.Stdout, "[TileServer] ", log.LUTC|log.Ldate|log.Ltime|log.Lshortfile|log.Lmicroseconds)

// Serve a single stylesheet via HTTP. Open view_tileserver.html in your browser
// to see the results.
// The created tiles are cached in an sqlite database (MBTiles 1.2 conform) so
// successive access a tile is much faster.
func TileserverWithCaching(engine string, layer_config map[string]string) {
	bind := fmt.Sprintf("0.0.0.0:%v", config.Port)
	if engine == "postgres" {
		t := maptiles.NewTileServerPostgres(config.Cache)
		for i := range layer_config {
			t.AddMapnikLayer(i, layer_config[i])
		}
		logger.Println("Connecting to postgres databas:")
		logger.Println("***", config.Cache)
		logger.Printf("Magic happens on port %s...", config.Port)
		logger.Fatal(http.ListenAndServe(bind, t))
	} else {
		t := maptiles.NewTileServerSqlite(config.Cache)
		for i := range layer_config {
			t.AddMapnikLayer(i, layer_config[i])
		}
		logger.Println("Connecting to sqlite3 database:")
		logger.Println("***", config.Cache)
		logger.Printf("Magic happens on port %s...", config.Port)
		logger.Fatal(http.ListenAndServe(bind, t))
	}

}

func init() {
	// TODO: add config file
	//flag.StringVar(&port, "p", "8080", "server port")
	//flag.StringVar(&engine, "e", "sqlite", "database engine [sqlite or postgres]")
	//flag.StringVar(&db_cache, "d", "tilecache.mbtiles", "tile cache database")
	flag.StringVar(&config_file, "c", "", "tile server config")
	flag.Parse()
	if engine != "sqlite" {
		if engine != "postgres" {
			logger.Fatal("Unsupported database engines")
		}
	}
}

func getConfig() {
	// check if file exists!!!
	if _, err := os.Stat(config_file); err == nil {
		file, err := ioutil.ReadFile(config_file)
		if err != nil {
			panic(err)
		}
		// config = make(map[string]string)
		err = json.Unmarshal(file, &config)
		if err != nil {
			logger.Println("error:", err)
		}
	} else {
		logger.Fatal("file not found")
	}
}

// Before uncommenting the GenerateOSMTiles call make sure you have
// the neccessary OSM sources. Consult OSM wiki for details.
func main() {
	getConfig()

	TileserverWithCaching(config.Engine, config.Layers)
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

