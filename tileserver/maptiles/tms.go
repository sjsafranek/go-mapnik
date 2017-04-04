package maptiles

import (
	"fmt"
	"net/http"
	"time"
)

// TMSErrorTile returns error response
func TMSErrorTile(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	http.Error(w, "Expecting /{layer}/{z}/{x}/{y}.png", http.StatusBadRequest)
	Ligneous.Info(fmt.Sprintf("%v %v %v [400]", r.RemoteAddr, r.URL.Path, time.Since(start)))
}

// TMSRootHandler root handler for tms server.
func TMSRootHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	var tree = `<?xml version="1.0" encoding="utf-8" ?>
				 <Services>
				 	<TileMapService title="` + SERVER_NAME + ` Tile Map Service" version="1.0" href="http:127.0.0.1/tms/1.0"/>
				 </Services>`
	status := SendXMLResponseFromString(tree, w, r)
	Ligneous.Info(fmt.Sprintf("%v %v %v [%v]", r.RemoteAddr, r.URL.Path, time.Since(start), status))
}

// TMSTileMaps returns list of available TileMaps.
func TMSTileMaps(start time.Time, lyrs []string, w http.ResponseWriter, r *http.Request) {
	var TileMaps = ``
	for _, lyr := range lyrs {
		TileMaps += `<TileMap title="` + lyr + `" srs="EPSG:4326" href="http:127.0.0.1:8080` + r.URL.Path + `/` + lyr + `"></TileMap>`
	}
	var tree = `<?xml version="1.0" encoding="utf-8" ?>
				 <TileMapService version="1.0" services="http:127.0.0.1:8080` + r.URL.Path + `">
				 	<Abstract></Abstract>
					<TileMaps>
						` + TileMaps + `
					</TileMaps>
				 </TileMapService>`
	status := SendXMLResponseFromString(tree, w, r)
	Ligneous.Info(fmt.Sprintf("%v %v %v [%v]", r.RemoteAddr, r.URL.Path, time.Since(start), status))
}

// TMSTileMap returns list of TileSets for layer.
func TMSTileMap(start time.Time, lyr string, source string, w http.ResponseWriter, r *http.Request) {
	var TileSets = ``
	for i := 0; i < 21; i++ {
		TileSets += `<TileSet
						href="` + fmt.Sprintf("http:127.0.0.1:8080%v/%v", r.URL.Path, i) + `"
						units-per-pixel="` + fmt.Sprintf("%v", unitsPerPixel(i)) + `"
						order="` + fmt.Sprintf("%v", i) + `">
					</TileSet>`
	}
	var tree = `<?xml version="1.0" encoding="utf-8" ?>
				 <TileMap version="1.0" services="http:127.0.0.1:8080` + r.URL.Path + `">
				 	<Title>` + lyr + `</Title>
                    <Source>` + source + `</Source>
					<Abstract></Abstract>
					<SRS>EPSG:4326</SRS>
					<BoundingBox minx="-180" miny="-90" maxx="180" max="90"></BoundingBox>
					<Origin x="-180" y="-90"></Origin>
					<TileFormat width="256" height="256" mime-type="image/png" extension="png"></TileFormat>
					<TileSets profile="global-geodetic">
						` + TileSets + `
					</TileSets>
				 </TileMap>`
	status := SendXMLResponseFromString(tree, w, r)
	Ligneous.Info(fmt.Sprintf("%v %v %v [%v]", r.RemoteAddr, r.URL.Path, time.Since(start), status))
}
