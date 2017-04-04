package maptiles

import (
	// "fmt"
	"net/http"
	"runtime"
	"time"
)

// PingHandler provides an api route for server health check
func PingHandler(w http.ResponseWriter, r *http.Request) {
	response := make(map[string]interface{})
	response["status"] = "ok"
	result := make(map[string]interface{})
	result["result"] = "Pong"
	response["data"] = result
	SendJsonResponseFromInterface(w, r, response)
}

// ServerProfileHandler returns basic server stats.
func ServerProfileHandler(startTime time.Time, w http.ResponseWriter, r *http.Request) {
	var data map[string]interface{}
	data = make(map[string]interface{})
	data["registered"] = startTime.UTC()
	data["uptime"] = time.Since(startTime).Seconds()
	data["num_cores"] = runtime.NumCPU()
	response := make(map[string]interface{})
	response["status"] = "ok"
	response["data"] = data
	SendJsonResponseFromInterface(w, r, response)
}
