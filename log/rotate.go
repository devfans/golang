package log

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Default log file flag
var LogFileFlag int = os.O_WRONLY|os.O_CREATE|os.O_APPEND

// Logger config
type LogConfig struct {
	// Logger level to use
	Level    Level
	MaxSize  uint   // max bytes in MB
	MaxFiles uint   // max log files
	Path     string // main log file path
}

// Provide logger writer instance, nil config will use os.Stderr instead
func (c *LogConfig) Writer() io.Writer {
	if c == nil || c.Path == "" {
		return os.Stderr
	}
	w, err := NewFileWriter(*c)
	if err != nil {
		fmt.Printf("Failed to create file writer, err %v\n", err)
		return os.Stderr
	}
	return w
}

// Create a file writer instance with a log config
func NewFileWriter(c LogConfig) (w *FileWriter, err error) {
	w = &FileWriter{LogConfig: c}
	err = w.Init()
	return
}

// FileWriter defines a file writer instance
type FileWriter struct {
	LogConfig     // Embed the config
	size, maxSize int
	file          *os.File
	ch            chan bool
}

// Initiate a file writer instance
func (w *FileWriter) Init() (err error) {
	w.Path = strings.TrimSpace(w.Path)
	if w.Path == "" {
		return fmt.Errorf("invalid log file path")
	}
	if !strings.HasSuffix(w.Path, ".log") {
		w.Path += ".log"
	}

	if w.MaxSize > 0 {
		w.maxSize = int(w.MaxSize << 20)
		if uint(w.maxSize>>20) != w.MaxSize {
			return fmt.Errorf("invalid max log file size")
		}
	}

	w.file, err = openFile(w.Path)
	if err == nil && w.MaxFiles > 0 {
		w.ch = make(chan bool, 1)
		w.ch <- true
		go w.removeLogs()
	}
	return
}

// Implement io.Writer interface for file writer
func (w *FileWriter) Write(p []byte) (n int, err error) {
	if w.maxSize > 0 && w.size+len(p) > w.maxSize {
		err = w.rotate()
		if err != nil {
			fmt.Printf("Failed to rotate log file, path: %s err: %v \n", w.Path, err)
			return
		}
	}
	n, err = w.file.Write(p)
	if err == nil && w.maxSize > 0 {
		w.size += len(p)
	}
	return
}

// Remove stale log files
func (w *FileWriter) removeLogs() {
	base := filepath.Base(w.Path)
	dir := filepath.Dir(w.Path)
	for range w.ch {
		list, err := os.ReadDir(dir)
		count := uint(0)
		if err == nil {
			for idx := len(list) - 1; idx >= 0; idx-- {
				name := list[idx].Name()
				if list[idx].Type().IsRegular() && strings.HasPrefix(name, base) {
					count++
					if count > w.MaxFiles && name != base {
						os.Remove(filepath.Join(dir, name))
					}
				}
			}
		}
	}
}

// Rotate will create new logger files and remove old ones when required
func (w *FileWriter) rotate() (err error) {
	if w.file != nil {
		w.file.Sync()
		w.file.Close()
		os.Rename(w.Path, fmt.Sprintf("%s%s", w.Path, time.Now().Format(time.RFC3339)))
		if w.ch != nil {
			select {
			case w.ch <- true:
			default:
			}
		}
	}
	w.file, err = openFile(w.Path)
	if err == nil {
		if info, err := w.file.Stat(); err == nil {
			w.size = int(info.Size())
		} else {
			w.size = 0
		}
	}
	return
}

// Close will try to close the file object
func (w *FileWriter) Close() (err error) {
	f := w.file
	if f != nil {
		return f.Close()
	}
	return nil
}

// openFile will try to open a log file with desired flags
func openFile(path string) (f *os.File, err error) {
	return os.OpenFile(path, LogFileFlag, 0664)
}
