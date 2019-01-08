package main

import (
	"fmt"
	"go/ast"
	"io"
	"reflect"
	"strings"
	"text/template"
)

type tpl struct {
	FieldName string
	FieldType string
	TagValue  string
}

var (
	fnMap = template.FuncMap{
		"lower": strings.ToLower,
		"enum": func(s string) string {
			arr := strings.Split(s, "|")
			return strings.Join(arr, ", ")
		},
	}

	requiredTpl = template.Must(template.New("required").Funcs(fnMap).Parse(`
	{{if eq .FieldType "int" -}}
	if t.{{.FieldName}} == 0 {
	{{else -}}
	if t.{{.FieldName}} == "" {
	{{end -}}
		return ApiError{
			HTTPStatus: http.StatusBadRequest, 
			Err: fmt.Errorf("{{lower .FieldName}} must me not empty"),
		}
	}
	`))

	minTpl = template.Must(template.New("min").Funcs(fnMap).Parse(`
	{{if eq .FieldType "int" -}}
	if t.{{.FieldName}} < {{.TagValue}} {
		return ApiError{
			HTTPStatus: http.StatusBadRequest, 
			Err: fmt.Errorf("{{lower .FieldName}} must be >= {{.TagValue}}"),
		}
	}
	{{else -}}
	if len(t.{{.FieldName}}) < {{.TagValue}} {
		return ApiError{
			HTTPStatus: http.StatusBadRequest, 
			Err: fmt.Errorf("{{lower .FieldName}} len must be >= {{.TagValue}}"),
		}
	}
	{{end}}	 
	`))

	maxTpl = template.Must(template.New("max").Funcs(fnMap).Parse(`
	if t.{{.FieldName}} > {{.TagValue}} {
		return ApiError{
			HTTPStatus: http.StatusBadRequest, 
			Err: fmt.Errorf("{{lower .FieldName}} must be <= {{.TagValue}}"),
		}
	}	 
	`))

	enumTpl = template.Must(template.New("enum").Funcs(fnMap).Parse(`
	if !strings.Contains("{{.TagValue}}", t.{{.FieldName}}) {
		return ApiError{
			HTTPStatus: http.StatusBadRequest, 
			Err: fmt.Errorf("{{lower .FieldName}} must be one of [{{enum .TagValue}}]"),
		} 
	}
	`))

	defaultTpl = template.Must(template.New("default").Parse(`
	{{if eq .FieldType "int" -}}
	if t.{{.FieldName}} == 0 {
	{{else -}}
	if t.{{.FieldName}} == "" {
	{{end -}}
		t.{{.FieldName}} = "{{.TagValue}}"
	}
	`))

	bindTpl = template.Must(template.New("bind").Funcs(fnMap).Parse(`
	{{if eq .FieldType "int" -}} 
	{{.FieldName}}, err := strconv.Atoi(q.Get("{{.TagValue}}")) 
	if err != nil {
		return ApiError{
			HTTPStatus: http.StatusBadRequest,
			Err: fmt.Errorf("{{lower .FieldName}} must be int"),
		}
	}
	t.{{.FieldName}} = {{.FieldName}}
	{{else -}} 
	t.{{.FieldName}} = q.Get("{{.TagValue}}") 
	{{end}}
	`))
)

func getTagValues(tag string) map[string]string {
	values := make(map[string]string, 6)
	// split tag value to kv by , and =
	arr := strings.Split(tag, ",")
	for _, value := range arr {
		arr := strings.Split(value, "=")
		if len(arr) == 1 {
			values[arr[0]] = ""
		} else {
			values[arr[0]] = arr[1]
		}
	}
	return values
}

func validateField(w io.Writer, t *ast.TypeSpec, fname, ftype, tag string) {
	values := getTagValues(tag)

	// order of validation
	templates := [...]struct {
		name string
		tpl  *template.Template
	}{
		{"default", defaultTpl},
		{"required", requiredTpl},
		{"min", minTpl},
		{"max", maxTpl},
		{"enum", enumTpl},
	}

	for _, info := range templates {
		if val, ok := values[info.name]; ok {
			fmt.Printf("generating validation code for %s.%s [%s]\n", t.Name.Name, fname, info.name)
			data := &tpl{FieldName: fname, FieldType: ftype, TagValue: val}
			info.tpl.Execute(w, data)
		}
	}
}

func validateStruct(w io.Writer, t *ast.TypeSpec, s *ast.StructType) {
	needGenerate := false
	type fieldInfo struct {
		typ string
		tag string
	}
	fields := make(map[string]fieldInfo)

	for _, field := range s.Fields.List {
		fname := field.Names[0].Name
		typ, ok := field.Type.(*ast.Ident)
		if !ok {
			fmt.Printf("SKIP %s.%s %T is not simple type like as int, string\n", t.Name.Name, fname, field.Type)
			return
		}
		ftype := typ.Name

		if field.Tag != nil {
			tag := reflect.StructTag(field.Tag.Value[1 : len(field.Tag.Value)-1])
			tagValue, ok := tag.Lookup("apivalidator")
			needGenerate = needGenerate || ok
			if ok {
				fields[fname] = fieldInfo{ftype, tagValue}
			}
		}
	}

	if needGenerate {
		fmt.Fprintf(w, "func (t %s) validate() error {", t.Name.Name)
		for k, v := range fields {
			validateField(w, t, k, v.typ, v.tag)
		}
		fmt.Fprintln(w, "return nil")
		fmt.Fprintln(w, "}")
		fmt.Fprintln(w)
	}
}

func bindField(w io.Writer, fname, ftype, tag string) {
	values := getTagValues(tag)

	paramname, ok := values["paramname"]
	if !ok || paramname == "" {
		paramname = fname
	}
	if paramname == "-" {
		return
	}

	data := &tpl{FieldName: fname, FieldType: ftype, TagValue: strings.ToLower(paramname)}
	bindTpl.Execute(w, data)
}

func bindStruct(w io.Writer, t *ast.TypeSpec, s *ast.StructType) {
	needGenerate := false
	type fieldInfo struct {
		typ string
		tag string
	}
	fields := make(map[string]fieldInfo)

	for _, field := range s.Fields.List {
		fname := field.Names[0].Name
		typ, ok := field.Type.(*ast.Ident)
		if !ok {
			fmt.Printf("SKIP %s.%s %T is not simple type like as int, string\n", t.Name.Name, fname, field.Type)
			return
		}
		ftype := typ.Name
		if field.Tag != nil {
			tag := reflect.StructTag(field.Tag.Value[1 : len(field.Tag.Value)-1])
			tagValue, ok := tag.Lookup("apivalidator")
			needGenerate = needGenerate || ok
			fields[fname] = fieldInfo{ftype, tagValue}
		}
	}

	if needGenerate {
		fmt.Printf("generating bind code for %s\n", t.Name.Name)
		fmt.Fprintf(w, "func (t *%s) bind(q url.Values) error {", t.Name.Name)
		for k, v := range fields {
			bindField(w, k, v.typ, v.tag)
		}
		fmt.Fprintln(w, "return nil")
		fmt.Fprintln(w, "}")
		fmt.Fprintln(w)
	}
}
