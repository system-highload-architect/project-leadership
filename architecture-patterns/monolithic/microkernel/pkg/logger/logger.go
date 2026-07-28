package logger

import "log"

var (
	Info  = log.New(log.Writer(), "[INFO] ", log.LstdFlags)
	Error = log.New(log.Writer(), "[ERROR] ", log.LstdFlags)
)
