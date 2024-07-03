package logger

import (
	"fmt"
	"log"
	"os"
)

type ConsoleLogger struct {
	debug *log.Logger
	info  *log.Logger
	warn  *log.Logger
	error *log.Logger
	panic *log.Logger
}

func NewConsoleLogger(flags int) *ConsoleLogger {
	return &ConsoleLogger{
		debug: log.New(os.Stdout, "DEBUG: ", flags),
		info:  log.New(os.Stdout, "INFO: ", flags),
		warn:  log.New(os.Stdout, "WARN: ", flags),
		error: log.New(os.Stdout, "ERROR: ", flags),
		panic: log.New(os.Stdout, "PANIC: ", flags),
	}
}

func (l *ConsoleLogger) Debug(v ...any) {
	l.debug.Println(v...)
}

func (l *ConsoleLogger) Info(v ...any) {
	l.info.Println(v...)
}

func (l *ConsoleLogger) Warn(v ...any) {
	fmt.Print("\033[0;33m")
	l.warn.Println(v...)
	fmt.Print("\033[m")
}

func (l *ConsoleLogger) Error(v ...any) {
	fmt.Print("\033[0;31m")
	l.error.Println(v...)
	fmt.Print("\033[m")
}

func (l *ConsoleLogger) Panic(v ...any) {
	defer fmt.Print("\033[m")
	fmt.Print("\033[0;31m")
	l.panic.Panic(v...)
}
