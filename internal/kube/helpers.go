package kube

import "errors"

func compositeErr(errs ...error) error {
	s := ""
	n := 0
	for i := range errs {
		if errs[i] != nil {
			if n == 1 {
				s = "composite error: [" + s
			}
			if n > 0 {
				s += ", "
			}
			s += errs[i].Error()
			n += 1
		}
	}
	if n > 0 {
		if n > 1 {
			s += "]"
		}
		return errors.New(s)
	}
	return nil
}
