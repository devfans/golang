package log

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"runtime/debug"
	"sync"
	"time"
)

// Log levels
const (
	TRACE Level = iota
	DEBUG
	VERBOSE
	INFO
	WARN
	ERROR
)

var (
	Levels = [6]string{"TRACE", "DEBUG", "VERBOSE", "INFO", "WARN", "ERROR"}
)

func stringifyLevel(level Level) string {
	return Levels[level]
}

func ParseLevel(target string) Level {
	for level, str := range Levels {
		if str == target {
			return Level(level)
		}
	}
	return INFO
}

var (
	// Global handles
	Root *Logger
	Output func(Level, string)
	Log func(Level, ...interface{})
	Logf func(Level, string, ...interface{})
	Json func(Level, interface{})
	Dump func(Level, interface{})
	Trace, Debug, Verbose, Info, Warn, Error Handle
	discard = func(string, ...interface{}) {}
)

func init() {
	Init(nil)
}

func Init(config *LogConfig) {
	Root = NewLogger(config)
	Root.SetLevel(ParseLevel(os.Getenv("LOG_LEVEL")))
	applyGlobalHanldes()
}

func SetLevel(target Level) {
	Root.SetLevel(target)
	applyGlobalHanldes()
}

func applyGlobalHanldes() {
	Output  = Root.Output
	Log     = Root.Log
	Logf    = Root.Logf
	Trace   = Root.Trace
	Debug   = Root.Debug
	Verbose = Root.Verbose
	Info    = Root.Info
	Warn    = Root.Warn
	Error   = Root.Error
	Json    = Root.Json
	Dump    = Root.Dump
}

func Stringify(arg interface{}) string {
	switch v := arg.(type) {
	case string:
		return v
	default:
		return fmt.Sprintf("%v", arg)
	}
}

func Format(msg string, args ...interface{}) string {
	count := len(args)
	for i := 1; i < count; i+=2 {
		msg = fmt.Sprintf("%s	%s=%s", msg, Stringify(args[i-1]), Stringify(args[i]))
	}
	if count & 1 == 1 {
		msg = fmt.Sprintf("%s	%s=", msg, args[count-1])
	}
	return msg
}


func GetStackInfo() string {
	info := debug.Stack()
	start := 0
	count := 0
	for i, token := range info {
		if token == 0x0A {
			count++
			if count == 7 {
				if i < len(info) {
					start = i+1
				}
				break
			}
		}
	}
	return string(info[start:])
}

func Fatal(msg string, args ...interface{}) {
	Output(ERROR, fmt.Sprintf("Fatal: %s\n%s", Format(msg, args...), GetStackInfo()))
	os.Exit(2)
}

type Handle func(string, ...interface{})
type Level int

type Logger struct {
	level Level
	Trace, Debug, Verbose, Info, Warn, Error Handle
	writer io.Writer
	sync.Mutex
}

func NewLogger(config *LogConfig) *Logger {
	logger := &Logger {
		writer: config.Writer(),
	}
	// Always set level as info for new logger
	logger.SetLevel(INFO)
	return logger
}

const TimeFormat = "2006-01-02 15:04:05.000 MST"
func (l *Logger) write(level string, msg string, args ...interface{}) {
	count := len(args)
	for i := 1; i < count; i+=2 {
		msg = fmt.Sprintf("%s	%s=%s", msg, Stringify(args[i-1]), Stringify(args[i]))
	}
	if count & 1 == 1 {
		msg = fmt.Sprintf("%-5s %s %s	%s=\n", level, time.Now().Format(TimeFormat), msg, args[count-1])
	} else {
		msg = fmt.Sprintf("%-5s %s %s\n", level, time.Now().Format(TimeFormat), msg)
	}
	l.Lock()
	l.writer.Write([]byte(msg))
	l.Unlock()
}

func (l *Logger) Fatal(msg string, args ...interface{}) {
	Output(ERROR, fmt.Sprintf("Fatal: %s\n%s", Format(msg, args...), GetStackInfo()))
	os.Exit(2)
}

func (l *Logger) Output(level Level, msg string) {
	if level < l.level { return }
	l.Write([]byte(msg), true)
}

func (l *Logger) Log(level Level, args ...interface{}) {
	if level < l.level { return }
	l.Write([]byte(Stringify(args)), true)
}

func (l *Logger) Logf(level Level, msg string, args ...interface{}) {
	if level < l.level { return }
	msg = fmt.Sprintf("%-5s %s %s\n", stringifyLevel(level), time.Now().Format(TimeFormat), msg)
	l.Write([]byte(fmt.Sprintf(msg, args...)), false)
}

func (l *Logger) Json(level Level, arg interface{}) {
	if level < l.level { return }
	bytes, err := json.Marshal(arg)
	if err != nil {
		bytes = []byte(err.Error())
	}
	l.Write(bytes, true)
}

func (l *Logger) Dump(level Level, arg interface{}) {
	if level < l.level { return }
	bytes, err := json.MarshalIndent(arg, "", "  ")
	if err != nil {
		bytes = []byte(err.Error())
	}
	l.Write(bytes, true)
}

func (l *Logger) Write(bytes []byte, newline bool) {
	l.Lock()
	l.writer.Write(bytes)
	if newline {
		l.writer.Write([]byte("\n"))
	}
	l.Unlock()
}

func (l *Logger) wrap(level string) Handle {
	return func(msg string, args ...interface{}) { l.write(level, msg, args...) }
}

func (l *Logger) SetLevel(target Level) {
	if target < 0 || target > 5 {
		target = INFO
		l.Output(ERROR, "Invalid log level, will use INFO by default.")
	}
	l.level = target
	handles := []*Handle{&l.Trace, &l.Debug, &l.Verbose, &l.Info, &l.Warn, &l.Error}
	for i, handle := range handles {
		if Level(i) >= target {
			*handle = l.wrap(stringifyLevel(Level(i)))
		} else {
			*handle = discard
		}
	}
}

