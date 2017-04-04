package maptiles

import (
	"encoding/json"
	"net/http"
)

// SendJsonResponseFromByte Sends http json response from byte.
func SendJsonResponseFromByte(content []byte, w http.ResponseWriter, r *http.Request) int {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Write(content)
	return 200
}

// SendJsonResponseFromString Sends http json response from string.
func SendJsonResponseFromString(content string, w http.ResponseWriter, r *http.Request) int {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Write([]byte(content))
	return 200
}

// SendJsonResponseFromInterface sends http json response from inteface.
func SendJsonResponseFromInterface(w http.ResponseWriter, r *http.Request, data interface{}) int {
	js, err := json.Marshal(data)
	var status int
	if nil != err {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		status = 500
	} else {
		status = SendJsonResponseFromByte(js, w, r)
	}
	return status
}

// SendXMLResponseFromString sends http xml response from string.
func SendXMLResponseFromString(content string, w http.ResponseWriter, r *http.Request) int {
	w.Header().Set("Content-Type", "text/xml")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Write([]byte(content))
	return 200
}
