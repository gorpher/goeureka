package goeureka

import "fmt"

type CustomError string

const (
	ErrorNotFoundService CustomError = "没有发现服务"
)

func (c CustomError) Error() string {
	return string(c)
}

func (c CustomError) Wrap(b error) error {
	return fmt.Errorf("%s %v", c.Error(), b)
}
