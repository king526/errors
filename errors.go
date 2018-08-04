// Package errors provides simple error handling primitives.
//
// The traditional error handling idiom in Go is roughly akin to
//
//     if err != nil {
//             return err
//     }
//
// which applied recursively up the call stack results in error reports
// without context or debugging information. The errors package allows
// programmers to add context to the failure path in their code in a way
// that does not destroy the original value of the error.
//
// Adding context to an error
//
// The errors.Wrap function returns a new error that adds context to the
// original error by recording a stack trace at the point Wrap is called,
// and the supplied message. For example
//
//     _, err := ioutil.ReadAll(r)
//     if err != nil {
//             return errors.Wrap(err, "read failed")
//     }
//
// If additional control is required the errors.WithStack and errors.WithMessage
// functions destructure errors.Wrap into its component operations of annotating
// an error with a stack trace and an a message, respectively.
//
// Retrieving the cause of an error
//
// Using errors.Wrap constructs a stack of errors, adding context to the
// preceding error. Depending on the nature of the error it may be necessary
// to reverse the operation of errors.Wrap to retrieve the original error
// for inspection. Any error value which implements this interface
//
//     type causer interface {
//             Cause() error
//     }
//
// can be inspected by errors.Cause. errors.Cause will recursively retrieve
// the topmost error which does not implement causer, which is assumed to be
// the original cause. For example:
//
//     switch err := errors.Cause(err).(type) {
//     case *MyError:
//             // handle specifically
//     default:
//             // unknown error
//     }
//
// causer interface is not exported by this package, but is considered a part
// of stable public API.
//
// Formatted printing of errors
//
// All error values returned from this package implement fmt.Formatter and can
// be formatted by the fmt package. The following verbs are supported
//
//     %s    print the error. If the error has a Cause it will be
//           printed recursively
//     %v    see %s
//     %+v   extended format. Each Frame of the error's StackTrace will
//           be printed in detail.
//
// Retrieving the stack trace of an error or wrapper
//
// New, Errorf, Wrap, and Wrapf record a stack trace at the point they are
// invoked. This information can be retrieved with the following interface.
//
//     type stackTracer interface {
//             StackTrace() errors.StackTrace
//     }
//
// Where errors.StackTrace is defined as
//
//     type StackTrace []Frame
//
// The Frame type represents a call site in the stack trace. Frame supports
// the fmt.Formatter interface that can be used for printing information about
// the stack trace of this error. For example:
//
//     if err, ok := err.(stackTracer); ok {
//             for _, f := range err.StackTrace() {
//                     fmt.Printf("%+s:%d", f)
//             }
//     }
//
// stackTracer interface is not exported by this package, but is considered a part
// of stable public API.
//
// See the documentation for Frame.Format for more details.
package errors

import (
	"fmt"
	"io"
	"regexp"
)

type status string

const (
	Unknown status = "Unknown"
)

var (
	statusRgx = regexp.MustCompile(`^[A-Z]\w*$`)
)

func NewStatus(s string) status {
	if !statusRgx.MatchString(s) {
		panic("invalid status string")
	}
	return status(s)
}

// New returns an error with the supplied message.
// New also records the stack trace at the point it was called.

func New(code status, msg ...interface{}) error {
	return &fundamental{
		code:  code,
		msg:   fmt.Sprint(msg...),
		stack: callers(),
	}
}

// Errorf formats according to a format specifier and returns the string
// as a value that satisfies error.
// Errorf also records the stack trace at the point it was called.
func Errorf(code status, format string, args ...interface{}) error {
	return &fundamental{
		code:  code,
		msg:   fmt.Sprintf(format, args...),
		stack: callers(),
	}
}

// fundamental is an error that has a message and a stack, but no caller.
type fundamental struct {
	code status
	msg  string
	*stack
}

func (f *fundamental) Error() string {
	s := string(f.code)
	if f.msg != "" {
		s += ":" + f.msg
	}
	return s

}

func (f *fundamental) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		if s.Flag('+') {
			io.WriteString(s, f.Error())
			f.stack.Format(s, verb)
			return
		}
		fallthrough
	case 's':
		io.WriteString(s, f.Error())
	case 'q':
		fmt.Fprintf(s, "%q", f.Error())
	}
}

// WithStack annotates err with a stack trace at the point WithStack was called.
// If err is nil, WithStack returns nil.

type withStack struct {
	withMessage
	*stack
}

func (w *withStack) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		if s.Flag('+') {
			io.WriteString(s, string(w.code)+":"+w.msg)
			w.stack.Format(s, verb)
			fmt.Fprintf(s, "\n%+v", w.Cause())
			return
		}
		fallthrough
	case 's':
		io.WriteString(s, w.Error())
	case 'q':
		fmt.Fprintf(s, "%q", w.Error())
	}
}

func WithStack(err error, code status, message ...interface{}) error {
	if err == nil {
		return nil
	}
	s := &withStack{
		stack: callers(),
	}
	s.cause = err
	s.code = code
	s.msg = fmt.Sprint(message...)
	return s
}

func WithStackf(err error, code status, format string, message ...interface{}) error {
	if err == nil {
		return nil
	}
	s := &withStack{
		stack: callers(),
	}
	s.cause = err
	s.code = code
	s.msg = fmt.Sprintf(format, message...)
	return s
}

func Wrap(err error, code status, message ...interface{}) error {
	if err == nil {
		return nil
	}
	return &withMessage{
		code:  code,
		cause: err,
		msg:   fmt.Sprint(message...),
	}
}

func Wrapf(err error, code status, format string, message ...interface{}) error {
	if err == nil {
		return nil
	}
	return &withMessage{
		code:  code,
		cause: err,
		msg:   fmt.Sprintf(format, message...),
	}
}

type withMessage struct {
	code  status
	cause error
	msg   string
}

func (w *withMessage) Error() string {
	s := string(w.code)
	if w.msg != "" {
		s += ":" + w.msg
	}
	return s + "; " + w.cause.Error()
}

func (w *withMessage) Cause() error { return w.cause }

func (w *withMessage) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		if s.Flag('+') {
			fmt.Fprintf(s, "%s:%s\n%+v\n", w.code, w.msg, w.Cause())
			return
		}
		fallthrough
	case 's', 'q':
		io.WriteString(s, w.Error())
	}
}

// Cause returns the underlying cause of the error, if possible.
// An error value has a cause if it implements the following
// interface:
//
//     type causer interface {
//            Cause() error
//     }
//
// If the error does not implement Cause, the original error will
// be returned. If the error is nil, nil will be returned without further
// investigation.

type causer interface {
	Cause() error
}

func Cause(err error) error {
	for err != nil {
		cause, ok := err.(causer)
		if !ok {
			break
		}
		err = cause.Cause()
	}
	return err
}

func StatusLine(err error) string {
	if err == nil {
		return ""
	}
	var s string
	for err != nil {
		switch t := err.(type) {
		case *fundamental:
			s += "." + string(t.code)
		case *withMessage:
			s += "." + string(t.code)
		case *withStack:
			s += "." + string(t.code)
		}
		cause, ok := err.(causer)
		if !ok {
			break
		}
		err = cause.Cause()
	}
	if s != "" {
		return s[1:]
	} else {
		return string(Unknown)
	}
}
