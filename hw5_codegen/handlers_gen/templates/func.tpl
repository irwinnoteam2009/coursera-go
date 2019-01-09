func (srv *{{.Recv}}) handle{{.Name}}(w http.ResponseWriter, r *http.Request) {
	// check method
	if !strings.Contains("{{.Method}}", r.Method) {
		body := response{Error: "bad method"}
		http.Error(w, body.String(), http.StatusNotAcceptable)
		return
	}
	{{if .Auth -}}
	// check authorization
	if r.Header.Get("X-Auth") != "100500" {
		body := response{Error: "unauthorized"}
		http.Error(w, body.String(), http.StatusForbidden)
		return
	}
	{{end -}}

	// bind values. If GET - get from query, if POST - get from form
	var values url.Values
	if r.Method == http.MethodGet {
		values = r.URL.Query()
	} else {
		r.ParseForm()
		values = r.Form
	}
	param := new({{.Param}})
	err := param.bind(values)

	if err != nil {
		err := err.(ApiError)
		body := response{Error: err.Error()}
		http.Error(w, body.String(), err.HTTPStatus)
		return
	}
	// validate
	err = param.validate()
	if err != nil {
		err := err.(ApiError)
		body := response{Error: err.Error()}
		http.Error(w, body.String(), err.HTTPStatus)
		return
	}	
	// 
	res, err := srv.{{.Name}}(context.Background(), *param)
	if err != nil {
		body := response{Error: err.Error()}
		if err, ok := err.(ApiError); ok {
			http.Error(w, body.String(), err.HTTPStatus)
			return
		}
		http.Error(w, body.String(), http.StatusInternalServerError)
		return
	}
	// OK
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	body := response{Error: "", Response: res}
	fmt.Fprintln(w, body.String())
}

