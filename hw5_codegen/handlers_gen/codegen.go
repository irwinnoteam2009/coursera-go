package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"log"
	"os"
	"strings"
	"text/template"
)

// код писать тут

var (
	outTpl = template.Must(template.ParseFiles("./templates/main.tpl"))
)

func main() {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, os.Args[1], nil, parser.ParseComments)
	if err != nil {
		log.Fatal(err)
	}

	out, _ := os.Create(os.Args[2])
	outTpl.Execute(out, node.Name.Name)

	ast.Inspect(node, func(n ast.Node) bool {
		switch t := n.(type) {
		case *ast.FuncDecl:
			processFunction(out, t)
		case *ast.TypeSpec:
			if currStruct, ok := t.Type.(*ast.StructType); ok {
				validateStruct(out, t, currStruct)
				bindStruct(out, t, currStruct)
			}
		}
		return true
	})

	for _, infos := range serveStructs {
		serveHTTP.Execute(out, infos)
	}
}

func processFunction(w io.Writer, fn *ast.FuncDecl) {
	if fn.Doc != nil && strings.HasPrefix(fn.Doc.Text(), "apigen:api") {
		comment := strings.TrimSpace(strings.TrimPrefix(fn.Doc.Text(), "apigen:api"))
		genHandler(w, fn, comment)
	}
}
