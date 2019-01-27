package main

import (
	"database/sql"
	"fmt"
	"reflect"
)

func scanStruct(r *sql.Rows) (interface{}, error) {
	a := make(map[string]interface{})

	columns, err := r.Columns()
	if err != nil {
		return nil, err
	}
	types, err := r.ColumnTypes()
	if err != nil {
		return nil, err
	}

	values := make([]interface{}, len(columns))
	for i := range values {
		values[i] = reflect.New(types[i].ScanType()).Interface()
	}

	if err := r.Scan(values...); err != nil {
		return nil, err
	}

	for i, col := range columns {
		switch v := values[i].(type) {
		case *int, *int8, *int16, *int32, *float32, *float64:
			a[col] = v
		case *sql.RawBytes:
			a[col] = nil
			if len(*v) != 0 {
				a[col] = string(*v)
			}
		default:
			fmt.Printf("%s: %T\n", col, v)
		}
	}

	return a, nil
}
