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

type methodInfo struct {
	// from comment
	URL    string `json:"url"`
	Auth   bool   `json:"auth,omitempty"`
	Method string `json:"method,omitempty"`
	// from reflection
	Recv  string `json:"recv,omitempty"`
	Name  string `json:"name,omitempty"`
	Param string `json:"param,omitempty"`
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
func (srv *{{.Recv}}
	`))

	serveStructs = map[string][]methodInfo{}
)

func getFuncReceiver(fn *ast.FuncDecl) string {
	var result string
	if fn.Recv != nil {
		switch t := fn.Recv.List[0].Type.(type) {
		case *ast.StarExpr:
			if x, ok := t.X.(*ast.Ident); ok {
				result = x.Name
			}
		case *ast.Ident:
			result = t.Name
		}
	}
	return result
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

	info.Name = fn.Name.Name
	info.Recv = getFuncReceiver(fn)
	info.Param = paramType

	addToServe(*info)

	fmt.Printf("%+v\n", info)
	funcTpl.Execute(w, info)
}

func addToServe(info methodInfo) {
	var infos []methodInfo
	var ok bool
	infos, ok = serveStructs[info.Recv]
	if !ok {
		infos = make([]methodInfo, 0)
	}
	infos = append(infos, info)
	serveStructs[info.Recv] = infos
}
