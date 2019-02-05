package main

import (
	"errors"
	"fmt"
)

var (
	errUnknownTable   = errors.New("unknown table")
	errRecordNotFound = errors.New("record not found")
	errSomethingWrong = errors.New("something wrong")
)

type typeError struct {
	field string
}

func (e *typeError) Error() string {
	return fmt.Sprintf("field %s have invalid type", e.field)
}

func newTypeError(field string) error {
	return &typeError{field: field}
}
