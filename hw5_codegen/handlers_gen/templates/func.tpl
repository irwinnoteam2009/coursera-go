func (srv *{{.Recv}}) handle{{.Name}}(w http.ResponseWriter, r *http.Request) {
	// check method
	if r.Method != "{{.Method}}" {
		body := response{Error: "bad error"}
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

	query := r.URL.Query()
	param := new({{.Param}})		
	// bind
	err := param.bind(query)
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
	fmt.Println(">>>>", body)
	fmt.Fprintln(w, body.String())
}

