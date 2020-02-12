package main

import (
	"log"
	"os"

	"github.com/logrusorgru/aurora"
)

var (
	Output = os.Stderr
	Flags  = log.Ltime | log.Lmicroseconds
)

var (
	logPanic *log.Logger
	logError *log.Logger
	logInfo  *log.Logger
	logDebug *log.Logger
)

func newLogger(prefix aurora.Value) *log.Logger {
	return log.New(Output, prefix.Bold().String(), Flags)
}

func ResetLoggers() {
	logPanic = newLogger(aurora.BgRed(aurora.White("PANIC")))
}
