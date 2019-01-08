package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"log"
	"os"
	"strings"
)

// код писать тут

func main() {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, os.Args[1], nil, parser.ParseComments)
	if err != nil {
		log.Fatal(err)
	}

	out, _ := os.Create(os.Args[2])
	fmt.Fprintln(out, "package ", node.Name.Name)
	fmt.Fprintln(out)
	fmt.Fprintln(out, `import "context"`)
	fmt.Fprintln(out, `import "fmt"`)
	fmt.Fprintln(out, `import "net/http"`)
	fmt.Fprintln(out, `import "net/url"`)
	fmt.Fprintln(out, `import "strconv"`)
	fmt.Fprintln(out, `import "strings"`)
	fmt.Fprintln(out)

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
}

func processFunction(w io.Writer, fn *ast.FuncDecl) {
	if fn.Doc != nil && strings.HasPrefix(fn.Doc.Text(), "apigen:api") {
		comment := strings.TrimSpace(strings.TrimPrefix(fn.Doc.Text(), "apigen:api"))
		genHandler(w, fn, comment)
	}
}
