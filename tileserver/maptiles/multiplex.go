package maptiles

// LayerMultiplex manages channels for tile requests.
type LayerMultiplex struct {
	layerChans map[string]chan<- TileFetchRequest
}

// NewLayerMultiplex creates LayerMultiplex struct.
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

// AddRenderer addes render for tile layer.
func (l *LayerMultiplex) AddRenderer(name string, stylesheet string) {
	l.layerChans[name] = NewTileRendererChan(stylesheet)
}

// AddSource manages tile requests.
func (l *LayerMultiplex) AddSource(name string, fetchChan chan<- TileFetchRequest) {
	l.layerChans[name] = fetchChan
}

// SubmitRequest submits tile request.
func (l LayerMultiplex) SubmitRequest(r TileFetchRequest) bool {
	c, ok := l.layerChans[r.Coord.Layer]
	if ok {
		c <- r
	} else {
		Ligneous.Warn("No such layer ", r.Coord.Layer)
	}
	return ok
}
