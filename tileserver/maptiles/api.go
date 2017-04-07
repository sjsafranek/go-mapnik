package maptiles

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

// NewTileLayer creates new tile layer.
func NewTileLayer(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	body, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	if nil != err {
		Ligneous.Critical(fmt.Sprintf("%v %v %v [500]", r.RemoteAddr, r.URL.Path, time.Since(start)))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	api_request := new(ApiRequest)
	err = json.Unmarshal(body, &api_request)
	if nil != err {
		Ligneous.Critical(fmt.Sprintf("%v %v %v [400]", r.RemoteAddr, r.URL.Path, time.Since(start)))
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	Ligneous.Info(fmt.Sprintf("%v", api_request))

	Ligneous.Info(fmt.Sprintf("%v %v %v [200]", r.RemoteAddr, r.URL.Path, time.Since(start)))

	SendJsonResponseFromString(`{"status": "ok"}`, w, r)
}
