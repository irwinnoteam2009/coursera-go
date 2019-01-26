package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

var (
	errUnknownTable   = errors.New("unknown table")
	errRecordNotFound = errors.New("record not found")
)

type response struct {
	Error    string      `json:"error,omitempty"`
	Response interface{} `json:"response,omitempty"`
}

// String ...
func (r *response) String() string {
	data, _ := json.Marshal(r)
	return string(data)
}

// ServeHTTP ...
func (db *DbExplorer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	fmt.Println(r.URL.Path)
	switch r.URL.Path {
	case "/":
		db.handlerGetTables(w, r)

	}
}

func handleError(w http.ResponseWriter, err error, code int) {
	resp := response{
		Error: err.Error(),
	}
	b, err := json.Marshal(resp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Error(w, string(b), code)
}

func handleResponse(w http.ResponseWriter, a interface{}) {
	resp := response{
		Response: a,
	}
	b, err := json.Marshal(resp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	fmt.Fprintln(w, string(b))
}

func (db *DbExplorer) handlerGetTables(w http.ResponseWriter, r *http.Request) {
	tables, err := db.getTableList()
	if err != nil {
		handleError(w, err, http.StatusInternalServerError)
		return
	}
	resp := struct {
		Tables []string `json:"tables"`
	}{
		Tables: tables,
	}
	handleResponse(w, resp)
}
