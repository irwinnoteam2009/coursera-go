package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)

type Row struct {
	XMLName   xml.Name `xml:"row"`
	ID        int      `xml:"id"`
	FirstName string   `xml:"first_name"`
	LastName  string   `xml:"last_name"`
	Age       int      `xml:"age"`
	About     string   `xml:"about"`
	Gender    string   `xml:"gender"`
}

func (r Row) Name() string {
	return r.FirstName + " " + r.LastName
}

type Rows []Row

func loadData(filename string) Rows {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		panic(err)
	}
	type Root struct {
		XMLName xml.Name `xml:"root"`
		Rows    []Row    `xml:"row"`
	}
	a := Root{}
	if err := xml.Unmarshal(data, &a); err != nil {
		panic(err)
	}
	rows := a.Rows
	return rows
}

func SearchServer(params SearchRequest) []User {
	rows := loadData("dataset.xml")
	fmt.Printf("%+v\n", params)
	// search
	var filtered Rows
	if params.Query == "" {
		filtered = rows
	} else {
		for i, row := range rows {
			if strings.Contains(row.Name(), params.Query) || strings.Contains(row.About, params.Query) {
				filtered = append(filtered, rows[i])
			}
		}
	}

	// order

	// map
	result := make([]User, len(filtered))
	for i, r := range filtered {
		user := User{
			Id:     r.ID,
			Name:   r.Name(),
			Age:    r.Age,
			Gender: r.Gender,
			About:  r.About,
		}
		result[i] = user
	}

	fmt.Println("result len:", len(result))
	return result
}

func testHandler(w http.ResponseWriter, r *http.Request) {
	writeResponse := func(w http.ResponseWriter, code int, a interface{}) {
		w.WriteHeader(code)
		if a != nil {
			d, _ := json.Marshal(a)
			w.Write(d)
		}
	}

	test := r.URL.Query().Get("query")
	switch test {
	case "token":
		writeResponse(w, http.StatusUnauthorized, nil)
	case "bad_order":
		writeResponse(w, http.StatusBadRequest, SearchErrorResponse{Error: ErrorBadOrderField})
	case "bad_json":
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, `{"status": 400`)
	case "bad_json2":
		w.WriteHeader(http.StatusBadRequest)
		io.WriteString(w, `{"status": 400`)
	case "bad_request":
		writeResponse(w, http.StatusBadRequest, SearchErrorResponse{Error: "some error here"})
	case "internal_error":
		writeResponse(w, http.StatusInternalServerError, nil)
	case "timeout":
		time.Sleep(2 * time.Second)
		writeResponse(w, http.StatusOK, nil)
	case "unknown":
		w.WriteHeader(302)
		w.Header().Set("Location", "")
	default:
		test = "ok"
	}

	if test != "ok" {
		return
	}

	orderField := r.URL.Query().Get("order_field")
	query := r.URL.Query().Get("query")
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")
	orderStr := r.URL.Query().Get("order_by")
	limit, _ := strconv.Atoi(limitStr)
	offset, _ := strconv.Atoi(offsetStr)
	orderBy, _ := strconv.Atoi(orderStr)

	params := SearchRequest{
		Limit:      limit,
		Offset:     offset,
		Query:      query,
		OrderField: orderField,
		OrderBy:    orderBy,
	}

	// // search
	users := SearchServer(params)
	writeResponse(w, http.StatusOK, users)
}

// код писать тут
func TestSearch(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(testHandler))
	defer ts.Close()

	for _, test := range [...]struct {
		name   string
		params SearchRequest
	}{
		{
			name:   "limit < 0",
			params: SearchRequest{Limit: -1},
		},
		{
			name:   "limit > 25",
			params: SearchRequest{Limit: 30},
		},
		{
			name:   "offset < 0",
			params: SearchRequest{Offset: -1},
		},
		{
			name:   "bad token",
			params: SearchRequest{Query: "token"},
		},
		{
			name:   "bad order",
			params: SearchRequest{Query: "bad_order"},
		},
		{
			name:   "bad json",
			params: SearchRequest{Query: "bad_json"},
		},
		{
			name:   "bad json2",
			params: SearchRequest{Query: "bad_json2"},
		},
		{
			name:   "internal error",
			params: SearchRequest{Query: "internal_error"},
		},
		{
			name:   "bad request",
			params: SearchRequest{Query: "bad_request"},
		},
		{
			name:   "timeout",
			params: SearchRequest{Query: "timeout"},
		},
		{
			name:   "test",
			params: SearchRequest{Query: "Rose"},
		},
		{
			name:   "unknown",
			params: SearchRequest{Query: "unknown"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			cli := SearchClient{URL: ts.URL}
			cli.FindUsers(test.params)
		})
	}
}
