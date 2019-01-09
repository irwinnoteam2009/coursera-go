package {{.}}

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type response struct {
	Error    string      `json:"error"`
	Response interface{} `json:"response,omitempty"`
}

func (r *response) String() string {
	data, _ := json.Marshal(r)
	return string(data)
}
