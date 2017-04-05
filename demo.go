package main

import (
	"fmt"
	// "math"
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

// degTorad converts degree to radians.
// func degTorad(deg float64) float64 {
// 	return deg * math.Pi / 180;
// }
//
// func deg2num(lat_deg float64, lon_deg float64, zoom int) (int, int) {
//     lat_rad := degTorad(lat_deg)
//     n := math.Pow(2.0, float64(zoom))
//     xtile := int((lon_deg + 180.0) / 360.0 * n)
//     ytile := int((1.0 - math.Log(math.Tan(lat_rad) + (1 / math.Cos(lat_rad))) / math.Pi) / 2.0 * n)
//     return xtile, ytile
// }

// func getTiles() {
// 	z := 5
// 	// upper right
// 	ur_tile_x, ur_tile_y := deg2num(float64(70), float64(16), z)
// 	// lower left
// 	ll_tile_x, ll_tile_y := deg2num(float64(35), float64(0), z)
// 	//
// 	fmt.Println("ur x", ur_tile_x)
// 	fmt.Println("ur y", ur_tile_y)
// 	fmt.Println("ll x", ll_tile_x)
// 	fmt.Println("ll y", ll_tile_y)
//
// 	for x := ll_tile_x; x < ur_tile_x+1; x++ {
// 		for y := ur_tile_y; y < ll_tile_y+1; y++ {
// 			fmt.Printf("/%v/%v/%v.png\n", z, x, y)
// 		}
// 	}
// }

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
