package main

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"io"
	"log"
	"net/http"
	"text/template"
)

type funcData struct {
	Recv   string
	Name   string
	URL    string
	Method string
	Auth   bool
	Param  string
}

type serveData struct {
	Recv    string
	URLs    []string
	Methods []string
}

var (
	funcTpl = template.Must(template.New("func").Parse(`
func (srv *{{.Recv}}) handle{{.Name}}(w http.ResponseWriter, r *http.Request) {
	// check method
	if r.Method != "{{.Method}}" {
		http.Error(w, "error": "bad method", http.StatusNotAcceptable)
	}
	{{if .Auth -}}
	// check authorization
	if r.Header.Get("X-Auth") != "100500" {
		http.Error(w, "error": "unauthorized", http.StatusForbidden)
	}
	{{end -}}

	query := r.URL.Query()
	param := new({{.Param}})		
	// bind
	err := param.bind(query)
	if err != nil {
		err := err.(ApiError)
		http.Error(w, err.Error(), err.HTTPStatus)
	}
	// validate
	err = param.validate()
	if err != nil {
		err := err.(ApiError)
		http.Error(w, err.Error(), err.HTTPStatus)
	}	
	// 
	res, err := srv.{{.Name}}(context.Background(), *param)
	if err != nil {
		err := err.(ApiError)
		http.Error(w, err.Error(), err.HTTPStatus)
	}
	// json
	data, err := json.Marshal(res)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	// OK
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintln(w, data)
}

	`))

	serveHTTP = template.Must(template.New("serve").Parse(`
	
	`))
)

type methodInfo struct {
	URL    string `json:"url"`
	Auth   bool   `json:"auth"`
	Method string `json:"method,omitempty"`
}

func genHandler(w io.Writer, fn *ast.FuncDecl, comment string) {
	// get method information
	info := &methodInfo{}
	if err := json.Unmarshal([]byte(comment), info); err != nil {
		log.Panicln(err)
	}
	if info.Method == "" {
		info.Method = http.MethodGet
	}

	// get receiver type
	var recvType string
	if fn.Recv != nil {
		switch t := fn.Recv.List[0].Type.(type) {
		case *ast.StarExpr:
			if x, ok := t.X.(*ast.Ident); ok {
				recvType = x.Name
			}
		case *ast.Ident:
			recvType = t.Name
		}
	}

	// fill params
	var paramType string
	if fn.Type.Params.List != nil {
		for _, p := range fn.Type.Params.List {
			switch t := p.Type.(type) {
			case *ast.Ident:
				paramType = t.Name
			case *ast.SelectorExpr:
				paramType = t.Sel.Name
			}
			if paramType == "Context" {
				continue
			}
		}
	}

	data := &funcData{
		Name:   fn.Name.Name,
		Recv:   recvType,
		URL:    info.URL,
		Method: info.Method,
		Auth:   info.Auth,
		Param:  paramType,
	}

	fmt.Printf("%+v\n", data)
	funcTpl.Execute(w, data)
}
