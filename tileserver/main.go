package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	log "github.com/cihub/seelog"
)

import "ligneous"
import "maptiles"

type Config struct {
	Cache  string            `json:"cache"`
	Engine string            `json:"engine"`
	Layers map[string]string `json:"layers"`
	Port   int               `json:"port"`
}

var (
	config Config
	// engine string
	// port string
	// db_cache string
	config_file   string
	print_version bool
	version       string = "0.1.3"
	//logger seelog.LoggerInterface
)

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
		log.Info("Connecting to postgres database:")
		log.Info("*** ", config.Cache)
		log.Info(fmt.Sprintf("Magic happens on port %v...", config.Port))
		srv := &http.Server{
			Addr:         bind,
			Handler:      t,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 10 * time.Second,
		}
		log.Error(srv.ListenAndServe())
		//log.Error(http.ListenAndServe(bind, t))
	} else {
		t := maptiles.NewTileServerSqliteMux(config.Cache)
		for i := range layer_config {
			t.AddMapnikLayer(i, layer_config[i])
		}
		log.Info("Connecting to sqlite3 database:")
		log.Info("*** ", config.Cache)
		log.Info(fmt.Sprintf("Magic happens on port %v...", config.Port))
		srv := &http.Server{
			Addr:         bind,
			Handler:      t.Router,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 10 * time.Second,
		}
		log.Error(srv.ListenAndServe())
		// log.Error(http.ListenAndServe(bind, t))
		//log.Error(http.ListenAndServe(bind, t.Router))
	}
	/*
		else {
			t := maptiles.NewTileServerSqlite(config.Cache)
			for i := range layer_config {
				t.AddMapnikLayer(i, layer_config[i])
			}
			log.Info("Connecting to sqlite3 database:")
			log.Info("*** ", config.Cache)
			log.Info(fmt.Sprintf("Magic happens on port %v...", config.Port))
			srv := &http.Server{
				Addr:         bind,
				Handler:      t,
				ReadTimeout:  5 * time.Second,
				WriteTimeout: 10 * time.Second,
			}
			log.Error(srv.ListenAndServe())
			//log.Error(http.ListenAndServe(bind, t.Mux))
		}
	*/
}

func init() {
	// TODO: add config file
	// flag.StringVar(&port, "p", "8080", "server port")
	// flag.StringVar(&engine, "e", "sqlite", "database engine [sqlite or postgres]")
	// flag.StringVar(&db_cache, "d", "tilecache.mbtiles", "tile cache database")
	flag.StringVar(&config_file, "c", "", "tile server config")
	flag.BoolVar(&print_version, "v", false, "version")
	flag.Parse()
	// if engine != "sqlite" {
	// 	if engine != "postgres" {
	// 		logger.Fatal("Unsupported database engines")
	// 	}
	// }
	if print_version {
		fmt.Println("TileServer", version)
		os.Exit(1)
	}

	logger, err := ligneous.InitLogger("TileServer")
	if nil != err {
		fmt.Println("Error starting logging")
		os.Exit(1)
	}
	log.UseLogger(logger)

}

func getConfig() {
	// check if file exists!!!
	if _, err := os.Stat(config_file); err == nil {

		file, err := ioutil.ReadFile(config_file)
		if err != nil {
			panic(err)
		}

		err = json.Unmarshal(file, &config)
		if err != nil {
			fmt.Println("error:", err)
			os.Exit(1)
		}

		if config.Engine != "sqlite" {
			if config.Engine != "postgres" {
				fmt.Println("Unsupported database engine")
				os.Exit(1)
			}
		}

		log.Debug(config)
	} else {
		fmt.Println("Config file not found")
		os.Exit(1)
	}
}

// Before uncommenting the GenerateOSMTiles call make sure you have
// the neccessary OSM sources. Consult OSM wiki for details.
func main() {
	getConfig()
	TileserverWithCaching(config.Engine, config.Layers)
}

// sudo su mapnik
// psql -d mbtiles -U mapnik -W
