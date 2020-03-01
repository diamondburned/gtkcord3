package log

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/logrusorgru/aurora"
)

var (
	Output = os.Stderr
	Flags  = log.Ltime | log.Lmicroseconds

	PrefixPanic  = "PANIC! "
	PrefixError  = "Error: "
	PrefixInfo   = "Info:  "
	PrefixDebug  = "Debug: "
	DebugGreyLvl = uint8(11)

	EnableDebug = false
)

var (
	logPanic *log.Logger
	logError *log.Logger
	logInfo  *log.Logger
	logDebug *log.Logger

	traceCtr uint64
)

func init() {
	ResetLoggers()
}

func newLogger(prefix aurora.Value) *log.Logger {
	return log.New(Output, prefix.Bold().String(), Flags)
}

func ResetLoggers() {
	logPanic = newLogger(aurora.BgRed(aurora.White(PrefixPanic)))
	logError = newLogger(aurora.Red(PrefixError))
	logInfo = newLogger(aurora.Blue(PrefixInfo))
	logDebug = newLogger(aurora.Gray(DebugGreyLvl, PrefixDebug))
}

// Trace, n is the argument to skip callers. 0 shows the location of the Trace
// function.
func Trace(n int) string {
	if !EnableDebug {
		return "<TRACE N/A>"
	}

	// i := atomic.AddUint64(&traceCtr, 1)
	i := 0

	_, file1, line1, _ := runtime.Caller(n + 1)
	_, file2, line2, _ := runtime.Caller(n + 2)
	_, file3, line3, _ := runtime.Caller(n + 3)

	file1 = filepath.Base(file1)
	file2 = filepath.Base(file2)
	file3 = filepath.Base(file3)

	return fmt.Sprintf(
		"%d ::: %s:%d > %s:%d > %s:%d >",
		i, file3, line3, file2, line2, file1, line1,
	)
}

func Infof(f string, v ...interface{}) {
	logInfo.Printf(f, v...)
}
func Infoln(v ...interface{}) {
	logInfo.Println(v...)
}
func Printf(f string, v ...interface{}) {
	logInfo.Printf(f, v...)
}
func Println(v ...interface{}) {
	logInfo.Println(v...)
}

func Debugf(f string, v ...interface{}) {
	if !EnableDebug {
		return
	}
	logDebug.Printf(f, v...)
}
func Debugln(v ...interface{}) {
	if !EnableDebug {
		return
	}
	logDebug.Println(v...)
}

func Errorf(f string, v ...interface{}) {
	logError.Printf(f, v...)
}
func Errorln(v ...interface{}) {
	logError.Println(v...)
}

func Panicf(f string, v ...interface{}) {
	logPanic.Panicf(f, v...)
}
func Panicln(v ...interface{}) {
	logPanic.Panicln(v...)
}
func Fatalf(f string, v ...interface{}) {
	logPanic.Fatalf(f, v...)
}
func Fatalln(v ...interface{}) {
	logPanic.Fatalln(v...)
}

func Benchmark(thing string) func() {
	now := time.Now()

	return func() {
		Debugln(thing, "took", time.Now().Sub(now))
	}
}

func BenchmarkLoop(thing string) (func(), func()) {
	now := time.Now()
	last := now
	duras := []time.Duration{}

	return func() {
			looped := time.Now()
			duras = append(duras, looped.Sub(last))
			last = looped
		},
		func() {
			Debugln(thing, "took cumulatively", time.Now().Sub(now), "individually:", duras)
		}
}
