package log

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"runtime/debug"
	"strings"
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
	// Log levels as string
	Levels = [6]string{"TRACE", "DEBUG", "VERBOSE", "INFO", "WARN", "ERROR"}
)

// Format level values into string representation
func stringifyLevel(level Level) string {
	return Levels[level]
}

// Parse level string
func ParseLevel(target string) Level {
	target = strings.ToUpper(target)
	for level, str := range Levels {
		if str == target {
			return Level(level)
		}
	}
	return INFO
}

var (
	// Global handles and root logger
	Root *Logger

	// Output raw strings
	Output func(Level, string)

	// Output any variables, like fmt.Println
	Println func(Level, ...interface{})

	// Output common logs in key=value format or string with args
	Log, Logf func(Level, string, ...interface{})

	// Dump args details as json string, Dump with indent
	Json, Dump func(Level, interface{})

	// Global handles for different levels
	Trace, Debug, Verbose, Info, Warn, Error Handle

	// default zero handle to discard messages
	discard = func(string, ...interface{}) {}
)

func init() {
	Init(nil)
}

// Initialize global logger
// Read default log level from config or global env variable LOG_LEVEL
func Init(config *LogConfig) {
	Root = NewLogger(config)
	level := ParseLevel(os.Getenv("LOG_LEVEL"))
	if config != nil {
		level = config.Level
	}
	Root.SetLevel(level)
	applyGlobalHanldes()
}

// Set logger levels for root logger
func SetLevel(target Level) {
	Root.SetLevel(target)
	applyGlobalHanldes()
}

func applyGlobalHanldes() {
	Output = Root.Output
	Log = Root.Log
	Logf = Root.Logf
	Println = Root.Println
	Trace = Root.Trace
	Debug = Root.Debug
	Verbose = Root.Verbose
	Info = Root.Info
	Warn = Root.Warn
	Error = Root.Error
	Json = Root.Json
	Dump = Root.Dump
}

// Format log string with args as key=value format
func Format(msg string, args ...interface{}) string {
	count := len(args)
	for i := 1; i < count; i += 2 {
		msg = fmt.Sprintf("%s	%s=%s", msg, Stringify(args[i-1]), Stringify(args[i]))
	}
	if count&1 == 1 {
		msg = fmt.Sprintf("%s	%s=", msg, args[count-1])
	}
	return msg
}

// Dump stack info with hiding logger internal calls
func GetStackInfo() string {
	info := string(debug.Stack())
	start := 0
	count := 0
	for i, token := range info {
		if token == 0x0A {
			count++
			if count == 7 {
				start = i + 1
				break
			}
		}
	}
	if start < len(info) {
		return info[start:]
	} else {
		return info
	}
}

// Fatal will exit the process after the log message is printed with stack info attached
func Fatal(msg string, args ...interface{}) {
	Output(ERROR, fmt.Sprintf("FATAL: %s\n%s", Format(msg, args...), GetStackInfo()))
	os.Exit(2)
}

// Log handle interface
type Handle func(string, ...interface{})

// Log level
type Level int

// Logger defines the logger instance
type Logger struct {
	// Current logger level
	level Level

	// Logger writer instance
	writer io.Writer
	// Logger writer lock to avoid race conditions
	sync.Mutex

	// Logger handles
	Trace, Debug, Verbose, Info, Warn, Error Handle
}

// Create new logger instance with an optional config
func NewLogger(config *LogConfig) *Logger {
	logger := &Logger{
		writer: config.Writer(),
	}
	// Always set level as info for new logger
	logger.SetLevel(INFO)
	return logger
}

// Time format used in loggers
const TimeFormat = "2006-01-02T15:04:05.000MST"

// Assemble the log message and write into output
func (l *Logger) write(level string, msg string, args ...interface{}) {
	count := len(args)
	for i := 1; i < count; i += 2 {
		msg = fmt.Sprintf("%s	%s=%s", msg, Stringify(args[i-1]), Stringify(args[i]))
	}
	if count&1 == 1 {
		msg = fmt.Sprintf("%-5s|%s %s	%s=\n", level, time.Now().Format(TimeFormat), msg, Stringify(args[count-1]))
	} else {
		msg = fmt.Sprintf("%-5s|%s %s\n", level, time.Now().Format(TimeFormat), msg)
	}
	l.Lock()
	l.writer.Write([]byte(msg))
	l.Unlock()
}

// Exit the process after the log message with stack info attached
func (l *Logger) Fatal(msg string, args ...interface{}) {
	Output(ERROR, fmt.Sprintf("FATAL: %s\n%s", Format(msg, args...), GetStackInfo()))
	os.Exit(2)
}

// Output a raw string with a custom level
func (l *Logger) Output(level Level, msg string) {
	if level < l.level {
		return
	}
	l.Write([]byte(msg), true)
}

// Output a log with custom level
func (l *Logger) Log(level Level, msg string, args ...interface{}) {
	if level < l.level {
		return
	}
	l.write(stringifyLevel(level), msg, args...)
}

// Output any args just like fmt.Println
func (l *Logger) Println(level Level, args ...interface{}) {
	if level < l.level {
		return
	}
	msg := fmt.Sprintf("%-5s|%s", stringifyLevel(level), time.Now().Format(TimeFormat))
	for _, arg := range args {
		msg = fmt.Sprintf("%s	%s", msg, stringify(arg))
	}
	l.Write([]byte(msg), true)
}

// Output a log message using string formatter with args
func (l *Logger) Logf(level Level, msg string, args ...interface{}) {
	if level < l.level {
		return
	}
	msg = fmt.Sprintf("%-5s|%s %s\n", stringifyLevel(level), time.Now().Format(TimeFormat), msg)
	l.Write([]byte(fmt.Sprintf(msg, args...)), false)
}

// Dump args as json
func (l *Logger) Json(level Level, arg interface{}) {
	if level < l.level {
		return
	}
	bytes, err := json.Marshal(arg)
	if err != nil {
		bytes = []byte(err.Error())
	}
	l.Write(bytes, true)
}

// Dump args as json with indent
func (l *Logger) Dump(level Level, arg interface{}) {
	if level < l.level {
		return
	}
	bytes, err := json.MarshalIndent(arg, "", "  ")
	if err != nil {
		bytes = []byte(err.Error())
	}
	l.Write(bytes, true)
}

// Write will write bytes with optional '\n' directly into output writer
func (l *Logger) Write(bytes []byte, newline bool) {
	l.Lock()
	l.writer.Write(bytes)
	if newline {
		l.writer.Write([]byte("\n"))
	}
	l.Unlock()
}

// Create a handle with a level string for the logger instance
func (l *Logger) wrap(level string) Handle {
	return func(msg string, args ...interface{}) { l.write(level, msg, args...) }
}

// Set a level for a logger instance
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
