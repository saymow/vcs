package errors

import (
	"runtime/debug"
)

func Check(err error) {
	if err != nil {
		Error(err.Error())
	}
}

func CheckFn(fn func() error) {
	err := fn()

	if err != nil {
		Error(err.Error())
	}
}

func Error(message string) {
	debug.PrintStack()
	panic(message)
}
