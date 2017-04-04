package maptiles

import (
	"fmt"
	"net/http"
	"runtime"
	"time"
)

// PingHandler provides an api route for server health check
func PingHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	response := make(map[string]interface{})
	response["status"] = "ok"
	result := make(map[string]interface{})
	result["result"] = "Pong"
	response["data"] = result
	status := SendJsonResponseFromInterface(w, r, response)
	Ligneous.Info(fmt.Sprintf("%v %v %v [%v]", r.RemoteAddr, r.URL.Path, time.Since(start), status))
}

// ServerProfileHandler returns basic server stats.
func ServerProfileHandler(startTime time.Time, w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	var data map[string]interface{}
	data = make(map[string]interface{})
	data["registered"] = startTime.UTC()
	data["uptime"] = time.Since(startTime).Seconds()
	data["num_cores"] = runtime.NumCPU()
	response := make(map[string]interface{})
	response["status"] = "ok"
	response["data"] = data
	status := SendJsonResponseFromInterface(w, r, response)
	Ligneous.Info(fmt.Sprintf("%v %v %v [%v]", r.RemoteAddr, r.URL.Path, time.Since(start), status))
}
