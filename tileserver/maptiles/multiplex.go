package maptiles

import (
	"tileserver/ligneous"
	log "github.com/cihub/seelog"
)

func init() {
	logger, _ := ligneous.InitLogger()
	log.UseLogger(logger)
}

type LayerMultiplex struct {
	layerChans map[string]chan<- TileFetchRequest
}

func NewLayerMultiplex() *LayerMultiplex {
	l := LayerMultiplex{}
	l.layerChans = make(map[string]chan<- TileFetchRequest)
	return &l
}

/*
func DefaultRenderMultiplex(defaultStylesheet string) *LayerMultiplex {
	l := NewLayerMultiplex()
	c := NewTileRendererChan(defaultStylesheet)
	l.layerChans[""] = c
	l.layerChans["default"] = c
	return l
}
*/

func (l *LayerMultiplex) AddRenderer(name string, stylesheet string) {
	l.layerChans[name] = NewTileRendererChan(stylesheet)
}

func (l *LayerMultiplex) AddSource(name string, fetchChan chan<- TileFetchRequest) {
	l.layerChans[name] = fetchChan
}

func (l LayerMultiplex) SubmitRequest(r TileFetchRequest) bool {
	c, ok := l.layerChans[r.Coord.Layer]
	if ok {
		c <- r
	} else {
		log.Warn("No such layer ", r.Coord.Layer)
	}
	return ok
}
