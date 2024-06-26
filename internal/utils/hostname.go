package utils

import (
	"os"
)

var _hostname string

func SetHostname() {
	if len(os.Getenv("POD_NAME")) > 0 {
		_hostname = os.Getenv("POD_NAME")
	} else {
		_hostname, _ = os.Hostname()
	}
}

func GetHostname() string {
	return _hostname
}
