package maptiles

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// MarshalJsonFromString mashals json from string.
// func MarshalJsonFromString(w http.ResponseWriter, r *http.Request, data string) ([]byte, error) {
// 	js, err := json.Marshal(data)
// 	if err != nil {
// 		message := fmt.Sprintf(" %v %v [500]", r.Method, r.URL.Path)
// 		Ligneous.Critical("[HttpServer] ", r.RemoteAddr, message)
// 		http.Error(w, err.Error(), http.StatusInternalServerError)
// 		return js, err
// 	}
// 	return js, nil
// }

// MarshalJsonFromInterface marshals interface into json.
func MarshalJsonFromInterface(w http.ResponseWriter, r *http.Request, data interface{}) ([]byte, error) {
	js, err := json.Marshal(data)
	if err != nil {
		Ligneous.Critical(fmt.Sprintf("[HttpServer] %v %v %v [500]", r.RemoteAddr, r.Method, r.URL.Path))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return js, err
	}
	return js, nil
}

// SendJsonResponse Sends http json response.
func SendJsonResponse(w http.ResponseWriter, r *http.Request, js []byte) {
	// Ligneous result
	Ligneous.Info(fmt.Sprintf("[HttpServer] %v %v %v [200]", r.RemoteAddr, r.Method, r.URL.Path))
	// set response headers
	w.Header().Set("Content-Type", "application/json")
	// allow cross domain AJAX requests
	w.Header().Set("Access-Control-Allow-Origin", "*")
	// write response content
	w.Write(js)
}

// SendJsonResponseFromInterface sends http json response from inteface.
func SendJsonResponseFromInterface(w http.ResponseWriter, r *http.Request, data interface{}) {
	js, err := MarshalJsonFromInterface(w, r, data)
	if nil == err {
		SendJsonResponse(w, r, js)
	}
}
