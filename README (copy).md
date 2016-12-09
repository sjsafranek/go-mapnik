# Tile Server
This application will serve MBTiles from a file named tiles.mbtiles. The route is http://localhost:8080/dbname/z/x/y

The HTML is a leaflet map that that loads tiles from http://localhost:8080/tiles/{z}/{x}/{y}

This project can be modified to server multiple tile layer if you copy the handler and assign it to another route making some small changes (dbname for starters). 

The structure of the server was taken from [Making a RESTful JSON API in Go](http://thenewstack.io/make-a-restful-json-api-go/)

The tileserver was built as a copy of the PHP version by [Bryan McBride](https://github.com/bmcbride/PHP-MBTiles-Server)



## TileServer

### Workspace setup
 - ``./install.sh``

### Build
 - ``make install``

### Run
 - ``./bin/tileserver -p 8888``



## Building MB Tiles with Mapbox Tilelive

### Resources
 - https://www.npmjs.com/package/tl
 - https://github.com/mapbox/tilelive

### Install Tilelive
 - ``npm install -g tl mbtiles tilelive-http``

### Usage
 - ``tl copy -z [zoom_min] -Z [zoom_max] -b "[WGS84 bounding box]" "[http tileserver address]" mbtiles://.//[DATABASENAME]/mbtiles``
 - Example: ``tl copy -z 0 -Z 15 -b "19.481506 49.050920 20.407791 49.319751" "http://a.tile.openstreetmap.org/{z}/{x}/{y}.png" mbtiles://./osm.mbtiles``

### Defaults
 - ``-b BBOX, --bounds BBOX        WGS84 bounding box  [-180,-85.0511,180,85.0511]``
 - ``-z ZOOM, --min-zoom ZOOM      Min zoom (inclusive)  [0]``
 - ``-Z ZOOM, --max-zoom ZOOM      Max zoom (inclusive)  [22]``




