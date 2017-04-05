package main

import (
	"fmt"
	"io/ioutil"

	"./tileserver/mapnik"
)

// Render a simple map of europe to a PNG file
func SimpleExample(map_file string) {
	m := mapnik.NewMap(1600, 1200)
	defer m.Free()
	m.Load(map_file)
	fmt.Println(m.SRS())
	// Perform a projection that is only necessary because stylesheet.xml
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

func main() {
	// lower left x & y
	// upper right x & y
	// height
	// width
	SimpleExample("sampledata/world_population/population.xml")
}

/*

export GOPATH="`pwd`"

*/
