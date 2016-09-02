package main

// This file contains various demo applications of the go-mapnik package

import (
	"fmt"
	"io/ioutil"
	"net/http"
	// "os"

	"flag"
	"log"

	"./mapnik"
	"./maptiles"
)

var (
	config map[string]string
	port string
)


// Render a simple map of europe to a PNG file
func SimpleExample(map_file string) {
	m := mapnik.NewMap(1600, 1200)
	defer m.Free()
	m.Load(map_file)
	fmt.Println(m.SRS())
	// Perform a projection that is only neccessary because stylesheet.xml
	// is using EPSG:3857 rather than WGS84
	p := m.Projection()
	ll := p.Forward(mapnik.Coord{0, 35})  // 0 degrees longitude, 35 degrees north
	ur := p.Forward(mapnik.Coord{16, 70}) // 16 degrees east, 70 degrees north
	m.ZoomToMinMax(ll.X, ll.Y, ur.X, ur.Y)
	blob, err := m.RenderToMemoryPng()
	if err != nil {
		fmt.Println(err)
		return
	}
	ioutil.WriteFile("mapnik.png", blob, 0644)
}

// Serve a single stylesheet via HTTP. Open view_tileserver.html in your browser
// to see the results.
// The created tiles are cached in an sqlite database (MBTiles 1.2 conform) so
// successive access a tile is much faster.
func TileserverWithCaching(layer_config map[string]string) {
	cache := "gomapnikcache.mbtiles"
	// os.Remove(cache)
	t := maptiles.NewTileServer(cache)
	for i := range layer_config {
		t.AddMapnikLayer(i, layer_config[i])
	}
	// CONFIG FILE
	log.Printf("Magic happens on port %s...", port)
	log.Fatal(http.ListenAndServe("0.0.0.0:"+port, t))
}


func init() {
	flag.StringVar(&port, "p", "8080", "server [port]")
	flag.Parse()
}

// Before uncommenting the GenerateOSMTiles call make sure you have
// the neccessary OSM sources. Consult OSM wiki for details.
func main() {
	// SimpleExample()
	//GenerateOSMTiles()
	config = make(map[string]string)
	config["default"] = "sampledata/stylesheet.xml"
	config["sample"] = "sampledata/stylesheet.xml"
	TileserverWithCaching(config)
}



/*

{
	"default": "sampledata/stylesheet.xml",
	"sample": "sampledata/stylesheet.xml"
}

*/



/*

https://github.com/mapbox/mbtiles-spec/blob/master/1.2/spec.md
https://github.com/sjsafranek/go-mapnik

apt-get install libmapnik-dev

export GOPATH="`pwd`"
go get -d github.com/mattn/go-sqlite3


go get -d github.com/sjsafranek/go-mapnik/mapnik

cd mapnik/
./configure.bash
cd ../
go run TileServer.go


*/

