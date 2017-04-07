package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"sort"
	"time"

	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"

	"flag"
)

// http://stackoverflow.com/questions/35964656/golang-how-to-concatenate-append-images-to-one-another

var (
	TILELAYER_URL string
	SAVEFILE string
	MIN_LAT float64
	MAX_LAT float64
	MIN_LNG float64
	MAX_LNG float64
	ZOOM int
	COOK bool
)

// Create a struct to deal with pixel
type Pixel struct {
	Point image.Point
	Color color.Color
}

// Decode image.Image's pixel data into []*Pixel
func DecodePixelsFromImage(img image.Image, offsetX, offsetY int) []*Pixel {
	pixels := []*Pixel{}
	for y := 0; y <= img.Bounds().Max.Y; y++ {
		for x := 0; x <= img.Bounds().Max.X; x++ {
			p := &Pixel{
				Point: image.Point{x + offsetX, y + offsetY},
				Color: img.At(x, y),
			}
			pixels = append(pixels, p)
		}
	}
	return pixels
}

var client = &http.Client{
	Timeout: time.Second * 5,
}

var tiles_map map[int][]image.Image

func init() {
	tiles_map = make(map[int][]image.Image)
}

// degTorad converts degree to radians.
func degTorad(deg float64) float64 {
	return deg * math.Pi / 180
}

// deg2num converts latlng to tile number
func deg2num(lat_deg float64, lon_deg float64, zoom int) (int, int) {
	lat_rad := degTorad(lat_deg)
	n := math.Pow(2.0, float64(zoom))
	xtile := int((lon_deg + 180.0) / 360.0 * n)
	ytile := int((1.0 - math.Log(math.Tan(lat_rad)+(1/math.Cos(lat_rad)))/math.Pi) / 2.0 * n)
	return xtile, ytile
}

// xyz
type xyz struct {
	x int
	y int
	z int
}

// GetTileNames
func GetTileNames(minlat, maxlat, minlng, maxlng float64, z int) []xyz {
	tiles := []xyz{}

	// upper right
	ur_tile_x, ur_tile_y := deg2num(maxlat, maxlng, z)
	// lower left
	ll_tile_x, ll_tile_y := deg2num(minlat, minlng, z)

	for x := ll_tile_x - 1; x < ur_tile_x+1; x++ {
		if x < 0 {
			x++
		}
		for y := ur_tile_y - 1; y < ll_tile_y+1; y++ {
			if y < 0 {
				y++
			}
			tiles = append(tiles, xyz{x, y, z})
		}
	}
	return tiles
}

// GetTilePngBytesFromUrl requests map tile png from url.
func GetTilePngBytesFromUrl(tile_url string) []byte {
	fmt.Println("GET", tile_url)

	// Just a simple GET request to the image URL
	// We get back a *Response, and an error
	res, err := client.Get(tile_url)
	if err != nil {
		fmt.Printf("Error http.Get -> %v\n", err)
		return []byte("")
	}

	// We read all the bytes of the image
	// Types: data []byte
	data, err := ioutil.ReadAll(res.Body)

	// You have to manually close the body, check docs
	// This is required if you want to use things like
	// Keep-Alive and other HTTP sorcery.
	defer res.Body.Close()

	if err != nil {
		fmt.Printf("Error ioutil.ReadAll -> %v\n", err)
		return []byte("")
	}

	// You can now save it to disk or whatever...
	// ioutil.WriteFile("TMP_TILE.png", data, 0644)
	// ioutil.WriteFile("TMP_TILE.png", data, 0666)

	return data
}

// BytesToPngImage converts bytes to png image struct.
func BytesToPngImage(b []byte) image.Image {
	img, err := png.Decode(bytes.NewReader(b))
	if nil != err {
		panic(err)
	}
	return img
}

// MergePngTiles combines png tiles into one image.
func MergePngTiles() image.Image {
	// Get bounds for new image.
	size := 256
	cols := 0
	rows := 0
	for i := range tiles_map {
		cols += size
		rows = len(tiles_map[i]) * size
	}

	// Sort columns.
	var columns []int
	for i := range tiles_map {
		columns = append(columns, i)
	}
	sort.Ints(columns)

	// Collect pixel data from each image.
	// Each image has a x-offset and Y-offset from the first.
	var pixelSum []*Pixel
	x := 0
	for _, i := range columns {
		y := 0
		for j := range tiles_map[i] {
			pixels := DecodePixelsFromImage(tiles_map[i][j], x, y)
			pixelSum = append(pixelSum, pixels...)
			y += size
		}
		x += size
	}

	// Set a new size for the new image equal to the max width
	// of bigger image and max height of two images combined.
	newRect := image.Rectangle{
		Min: image.Point{X: 0, Y: 0},
		Max: image.Point{X: cols, Y: rows},
	}

	// Create new image for final output.
	finImage := image.NewRGBA(newRect)

	// This is the cool part, all you have to do is loop through
	// each Pixel and set the image's color on the go.
	for _, px := range pixelSum {
		finImage.Set(
			px.Point.X,
			px.Point.Y,
			px.Color,
		)
	}
	draw.Draw(finImage, finImage.Bounds(), finImage, image.Point{0, 0}, draw.Src)

	return finImage
}

func savePng(filename string, img image.Image) {
	// Create a new file and write to it.
	out, err := os.Create(filename)
	if err != nil {
		panic(err)
		os.Exit(1)
	}
	err = png.Encode(out, img)
	if err != nil {
		panic(err)
		os.Exit(1)
	}
}

func main() {

	flag.StringVar(&TILELAYER_URL, "u", "http://localhost:8080/tms/1.0/population", "tile layer url")
	flag.StringVar(&SAVEFILE, "o", "output.png", "save png file")
	flag.Float64Var(&MIN_LAT, "minlat", -85, "min latitude")
	flag.Float64Var(&MAX_LAT, "maxlat", 85, "max latitude")
	flag.Float64Var(&MIN_LNG, "minlng", -175, "min longitude")
	flag.Float64Var(&MAX_LNG, "maxlng", 175, "max longitude")
	flag.IntVar(&ZOOM, "z", 3, "zoom")
	flag.BoolVar(&COOK, "c", false, "cook map tiles")
	flag.Parse()

	tiles := GetTileNames(MIN_LAT, MAX_LAT, MIN_LNG, MAX_LNG, ZOOM)

	cooked_tiles := 0

	for _, v := range tiles {
		tile_url := fmt.Sprintf("/%v/%v/%v.png", v.z, v.x, v.y)
		data := GetTilePngBytesFromUrl(TILELAYER_URL + tile_url)
		if !COOK {
			img := BytesToPngImage(data)
			tiles_map[v.x] = append(tiles_map[v.x], img)
		}
		cooked_tiles++
	}

	if !COOK {
		finalImage := MergePngTiles()
		savePng("./"+SAVEFILE, finalImage)
	}

	fmt.Println("Cooked tiles: ", cooked_tiles)
}

/*

export GOPATH="`pwd`"

*/
