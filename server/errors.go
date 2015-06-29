package server

import (
	"errors"
	"fmt"
)

type Errors []error

func (errs Errors) Len() int {
	n := 0
	for _, err := range errs {
		if err != nil {
			n++
		}
	}
	return n
}

func (errs Errors) Error() string {
	n := errs.Len()

	if n == 0 {
		return ""
	}
	if n == 1 {
		for _, err := range errs {
			if err != nil {
				return err.Error()
			}
		}
	}

	msg := fmt.Sprintf("%d errors:", n)
	for _, err := range errs {
		if err != nil {
			msg += "\n" + err.Error()
		}
	}
	return msg
}

func (errs Errors) Merge() error {
	n := errs.Len()

	if n == 0 {
		return nil
	}
	if n == 1 {
		for _, err := range errs {
			if err != nil {
				return err
			}
		}
	}

	return errors.New(errs.Error())
}
