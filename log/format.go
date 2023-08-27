package log

import (
	"fmt"
	"math/big"
	"reflect"
	"strconv"
	"time"
)

// formatValue formats a value for serialization
func Stringify(value interface{}) string {
	if value == nil {
		return "nil"
	}

	switch v := value.(type) {
	case time.Time:
		// Performance optimization: No need for escaping since the provided
		// timeFormat doesn't have any escape characters, and escaping is
		// expensive.
		return v.Format(TimeFormat)

	case *big.Int:
		// Big ints get consumed by the Stringer clause so we need to handle
		// them earlier on.
		if v == nil {
			return "<nil>"
		}
		return formatLogfmtBigInt(v)
	}
	value = formatShared(value)
	switch v := value.(type) {
	case bool:
		return strconv.FormatBool(v)
	case float32:
		return strconv.FormatFloat(float64(v), 'f', 3, 64)
	case float64:
		return strconv.FormatFloat(v, 'f', 3, 64)
	case int8:
		return strconv.FormatInt(int64(v), 10)
	case uint8:
		return strconv.FormatInt(int64(v), 10)
	case int16:
		return strconv.FormatInt(int64(v), 10)
	case uint16:
		return strconv.FormatInt(int64(v), 10)
	// Larger integers get thousands separators.
	case int:
		return FormatLogfmtInt64(int64(v))
	case int32:
		return FormatLogfmtInt64(int64(v))
	case int64:
		return FormatLogfmtInt64(v)
	case uint:
		return FormatLogfmtUint64(uint64(v))
	case uint32:
		return FormatLogfmtUint64(uint64(v))
	case uint64:
		return FormatLogfmtUint64(v)
	case string:
		return escapeString(v)
	default:
		return escapeString(fmt.Sprintf("%+v", value))
	}
}

// FormatLogfmtInt64 formats n with thousand separators.
func FormatLogfmtInt64(n int64) string {
	if n < 0 {
		return formatLogfmtUint64(uint64(-n), true)
	}
	return formatLogfmtUint64(uint64(n), false)
}

// FormatLogfmtUint64 formats n with thousand separators.
func FormatLogfmtUint64(n uint64) string {
	return formatLogfmtUint64(n, false)
}

func formatLogfmtUint64(n uint64, neg bool) string {
	// Small numbers are fine as is
	if n < 100000 {
		if neg {
			return strconv.Itoa(-int(n))
		} else {
			return strconv.Itoa(int(n))
		}
	}
	// Large numbers should be split
	const maxLength = 26

	var (
		out   = make([]byte, maxLength)
		i     = maxLength - 1
		comma = 0
	)
	for ; n > 0; i-- {
		if comma == 3 {
			comma = 0
			out[i] = ','
		} else {
			comma++
			out[i] = '0' + byte(n%10)
			n /= 10
		}
	}
	if neg {
		out[i] = '-'
		i--
	}
	return string(out[i+1:])
}

// formatLogfmtBigInt formats n with thousand separators.
func formatLogfmtBigInt(n *big.Int) string {
	if n.IsUint64() {
		return FormatLogfmtUint64(n.Uint64())
	}
	if n.IsInt64() {
		return FormatLogfmtInt64(n.Int64())
	}

	var (
		text  = n.String()
		buf   = make([]byte, len(text)+len(text)/3)
		comma = 0
		i     = len(buf) - 1
	)
	for j := len(text) - 1; j >= 0; j, i = j-1, i-1 {
		c := text[j]

		switch {
		case c == '-':
			buf[i] = c
		case comma == 3:
			buf[i] = ','
			i--
			comma = 0
			fallthrough
		default:
			buf[i] = c
			comma++
		}
	}
	return string(buf[i+1:])
}

// escapeString checks if the provided string needs escaping/quoting, and
// calls strconv.Quote if needed
func escapeString(s string) string {
	needsQuoting := false
	for _, r := range s {
		// We quote everything below " (0x34) and above~ (0x7E), plus equal-sign
		if r <= '"' || r > '~' || r == '=' {
			needsQuoting = true
			break
		}
	}
	if !needsQuoting {
		return s
	}
	return strconv.Quote(s)
}

func formatShared(value interface{}) (result interface{}) {
	defer func() {
		if err := recover(); err != nil {
			if v := reflect.ValueOf(value); v.Kind() == reflect.Ptr && v.IsNil() {
				result = "nil"
			} else {
				panic(err)
			}
		}
	}()

	switch v := value.(type) {
	case time.Time:
		return v.Format(TimeFormat)

	case error:
		return v.Error()

	case fmt.Stringer:
		return v.String()

	default:
		return v
	}
}