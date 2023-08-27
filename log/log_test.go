package log

import "testing"

func TestLogg(t *testing.T) {
	Info("Checking logging", "a", 1, "b", 2)
	SetLevel(DEBUG)
	Debug("Checking logging", "a", 1, "b", 2)
	Warn("Checking logging", "a", 1, "b", 2)
	SetLevel(WARN)
	Info("Checking logging", "a", 1, "b", 2)
	type Arg struct {
		B string
		C int
	}
	Error("Checking err", "err", new(Arg))
	Json(WARN, Arg{"test", 1})
	Dump(WARN, Arg{"xxx", 2})
	logger := NewLogger(&LogConfig{Path: "test.log"})
	logger.Error("Checking err", "err", new(Arg))
	logger.Json(ERROR, Arg{"test", 1})
	logger.Dump(ERROR, Arg{"xxx", 2})
	Logf(ERROR, "Check format %s %d ...", "x", 1)
	Fatal("Check fatal", "a", 1, "b", "xxx")
}
