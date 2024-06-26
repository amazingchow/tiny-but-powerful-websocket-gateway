package service

import "errors"

var (
	ErrExceedMaxConnNum = errors.New("maximum connections exceeded")
)
