package maptiles

import (
	"fmt"
	"net/http"
)

func TMSRootHandler(w http.ResponseWriter, r *http.Request) {
	var root = `<?xml version="1.0" encoding="utf-8" ?>
				 <Services>
				 	<TileMapService title="GoMapnik Tile Map Service" version="1.0" href="http:127.0.0.1/tms/1.0"/>
				 </Services>`
	fmt.Fprintf(w, root)
}

func TMSTileMaps(lyrs []string, w http.ResponseWriter, r *http.Request) {
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
	fmt.Fprintf(w, tree)
}

func TMSTileMap(lyr string, source string, w http.ResponseWriter, r *http.Request) {
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
	fmt.Fprintf(w, tree)
}
