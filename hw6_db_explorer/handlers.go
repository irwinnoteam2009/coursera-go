package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
)

const (
	paramLimit  = "limit"
	paramOffset = "offset"
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

type handlerFunc func(table, id string, w http.ResponseWriter, r *http.Request)

// ServeHTTP ...
func (db *DbExplorer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	tableHandlers := map[string]handlerFunc{
		http.MethodGet: db.handlerGetItemList,
		http.MethodPut: db.handlerAddItem,
	}

	itemsHandlers := map[string]handlerFunc{
		http.MethodGet:    db.handlerGetItem,
		http.MethodPost:   db.handlerUpdateItem,
		http.MethodDelete: db.handlerDeleteItem,
	}

	switch r.URL.Path {
	case "/":
		db.handlerGetTables(w, r)
	default:
		path := strings.Trim(r.URL.Path, "/")
		arr := strings.Split(path, "/")
		table := arr[0]
		var id string
		if len(arr) > 1 {
			id = arr[1]
		}

		if err := db.tableExists(table); err != nil {
			if err == errUnknownTable {
				handleError(w, err, http.StatusNotFound)
			} else {
				handleError(w, err, http.StatusInternalServerError)
			}
			return
		}

		var handler handlerFunc
		if len(arr) == 1 {
			handler = tableHandlers[r.Method]
		} else {
			handler = itemsHandlers[r.Method]
		}
		if handler != nil {
			handler(table, id, w, r)
		}
	}
}

func handleError(w http.ResponseWriter, err error, code int) {
	resp := response{
		Error: err.Error(),
	}
	// fmt.Println("handleError:", err)

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

func (db *DbExplorer) handlerGetItemList(table, id string, w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	sLimit := query.Get(paramLimit)
	sOffset := query.Get(paramOffset)
	limit, _ := strconv.Atoi(sLimit)
	offset, _ := strconv.Atoi(sOffset)

	data, err := db.getItemsList(table, limit, offset)
	if err != nil {
		handleError(w, err, http.StatusInternalServerError)
		return
	}

	resp := struct {
		Records []interface{} `json:"records"`
	}{
		Records: data,
	}
	handleResponse(w, resp)
}

func (db *DbExplorer) handlerAddItem(table, id string, w http.ResponseWriter, r *http.Request) {
	var a interface{}
	data, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		handleError(w, err, http.StatusInternalServerError)
		return
	}

	if err := json.Unmarshal(data, &a); err != nil {
		handleError(w, err, http.StatusInternalServerError)
		return
	}

	affected, err := db.createItem(table, a)
	pk := db.getPK(table)
	if err != nil || pk == "" {
		handleError(w, err, http.StatusInternalServerError)
		return
	}

	resp := map[string]int64{
		pk: affected,
	}
	handleResponse(w, resp)
}

func (db *DbExplorer) handlerGetItem(table, id string, w http.ResponseWriter, r *http.Request) {
	item, err := db.getItem(table, id)
	if err != nil {
		if err == errRecordNotFound {
			handleError(w, err, http.StatusNotFound)
		} else {
			handleError(w, err, http.StatusInternalServerError)
		}
		return
	}
	resp := struct {
		Record interface{} `json:"record"`
	}{
		Record: item,
	}
	handleResponse(w, resp)
}

func (db *DbExplorer) handlerUpdateItem(table, id string, w http.ResponseWriter, r *http.Request) {
	var a interface{}
	data, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		handleError(w, err, http.StatusInternalServerError)
		return
	}

	if err := json.Unmarshal(data, &a); err != nil {
		handleError(w, err, http.StatusInternalServerError)
		return
	}

	i, err := db.updateItem(table, id, a)
	if err != nil {
		if _, ok := err.(*typeError); ok {
			handleError(w, err, http.StatusBadRequest)
		} else {
			handleError(w, err, http.StatusInternalServerError)
		}
		return
	}

	resp := struct {
		Updated int64 `json:"updated"`
	}{
		Updated: i,
	}
	handleResponse(w, resp)
}

func (db *DbExplorer) handlerDeleteItem(table, id string, w http.ResponseWriter, r *http.Request) {
	i, err := db.deleteItem(table, id)
	if err != nil {
		handleError(w, err, http.StatusInternalServerError)
		return
	}

	resp := struct {
		Deleted int64 `json:"deleted"`
	}{
		Deleted: i,
	}
	handleResponse(w, resp)
}
