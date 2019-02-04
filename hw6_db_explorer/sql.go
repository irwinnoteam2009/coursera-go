package main

import (
	"database/sql"
	"fmt"
)

type myValue struct {
	valid bool
	value interface{}
}

func (m *myValue) Scan(v interface{}) error {
	switch v.(type) {
	case int:
		if value, ok := v.(int); ok {
			m.valid = true
			m.value = value
		}
	case int64:
		if value, ok := v.(int64); ok {
			m.valid = true
			m.value = value
		}
	case float64:
		if value, ok := v.(float64); ok {
			m.valid = true
			m.value = value
		}
	case []byte:
		if value, ok := v.([]byte); ok {
			m.valid = true
			m.value = string(value)
		}
	case string:
		if value, ok := v.(string); ok {
			fmt.Println(value)
		}
	case nil:
		m.valid = true
		m.value = nil
	}
	return nil
}

func scanStruct(r *sql.Rows) (interface{}, error) {
	a := make(map[string]interface{})

	columns, err := r.Columns()
	if err != nil {
		return nil, err
	}

	values := make([]interface{}, len(columns))
	for i := range values {
		values[i] = new(myValue)
	}

	if err := r.Scan(values...); err != nil {
		return nil, err
	}

	for i, col := range columns {
		v, ok := values[i].(*myValue)
		if !ok || !v.valid {
			continue
		}
		a[col] = v.value

		fmt.Printf("%s: %T %+v\n", col, values[i], values[i])
	}

	return a, nil
}
