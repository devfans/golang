package log

import "testing"

type Arg struct {
	B string
	C int
}

func (a *Arg) String() string {
	return "a string"
}

type B int
func (b B) Hex() string { return "b hex" }

func TestLogg(t *testing.T) {
	var b String
	Info("Checking logging", "a", 1, "b", 2)
	SetLevel(DEBUG)
	Debug("Checking logging", "a", 1, "b", 2)
	Warn("Checking logging", "a", 1, "b", 2)
	SetLevel(WARN)
	Info("Checking logging", "a", 1, "b", 2)
	Error("Checking err", "err", new(Arg), "hex", B(0), "nil", b, b)
	Logf(WARN, "Test format %s %v... %s", "xxx", 100)
	Json(WARN, Arg{"test", 1})
	Dump(WARN, Arg{"xxx", 2})
	logger := NewLogger(&LogConfig{Path: "test.log"})
	logger.Error("Checking err", "err", new(Arg))
	logger.Json(ERROR, Arg{"test", 1})
	logger.Dump(ERROR, Arg{"xxx", 2})
	Logf(ERROR, "Check format %s %d ...", "x", 1)

	
	Fatal("Check fatal", "a", 1, "b", "xxx")
}
