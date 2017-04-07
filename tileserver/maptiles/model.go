package maptiles

type ApiRequest struct {
	Method string        `json:"method"`
	Data   ApiReqestData `json:"data"`
}

type ApiReqestData struct {
	TileLayerSource string `json:"source"`
	TileLayerName   string `json:"name"`
}
