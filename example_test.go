package errors_test

import (
	"fmt"
	"testing"

	"github.com/king526/errors"
)

var (
	e0 = errors.NewStatus("Error0")
	e1 = errors.NewStatus("Error1")
	e2 = errors.NewStatus("Error2")
)

func Test_format(t *testing.T) {
	err := errors.New(e0, "EOF", 555)
	err = errors.WithStack(err, e1)
	err = errors.Wrap(err, e2)
	fmt.Println(errors.Cause(err))
	fmt.Println(err)
	fmt.Println(err.Error())
	fmt.Println(errors.StatusLine(err))
	fmt.Printf("%+v\n", err)
}
