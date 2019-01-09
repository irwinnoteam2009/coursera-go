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

type serveInfo struct {
	Recv    string
	Methods []methodInfo
}

var (
	funcTpl   = template.Must(template.ParseFiles("./templates/func.tpl"))
	serveHTTP = template.Must(template.New("serve").Parse(`
func (srv *{{.Recv}}) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	{{range .Methods -}}
	case "{{.URL}}": srv.handle{{.Name}}(w, r)
	{{end -}}
	default: 
	  	body := response{Error: "unknown method"}
		http.Error(w, body.String(), http.StatusNotFound)
	}
}
	`))

	serveStructs = map[string]serveInfo{}
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
	// if method not specified, it can use GET and POST
	if info.Method == "" {
		info.Method = http.MethodGet + "|" + http.MethodPost
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
	serv, ok := serveStructs[info.Recv]
	if !ok {
		infos = make([]methodInfo, 0)
	} else {
		infos = serv.Methods
	}
	infos = append(infos, info)
	serveStructs[info.Recv] = serveInfo{info.Recv, infos}
}
