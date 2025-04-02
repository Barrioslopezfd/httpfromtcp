package headers

import (
	"bytes"
	"fmt"
	"slices"
	"strings"
)

const (
	SEPARATOR = ":"
	CRLF      = "\r\n"
)

type field_line struct {
	field_name  string
	field_value string
}

type Headers map[string]string

func NewHeaders() Headers {
	return make(Headers)
}

func (h Headers) Parse(data []byte) (n int, done bool, err error) {
	idx := bytes.Index(data, []byte(CRLF))
	if idx == -1 {
		return 0, false, nil
	}
	if idx == 0 {
		return 2, true, nil
	}
	fl, parsed, err := parseHeader(data[:idx])
	if err != nil {
		return 0, false, err
	}
	val, ok := h[fl.field_name]
	if ok {
		if !strings.Contains(val, fl.field_value) {
			val = val + ", " + fl.field_value
			h[fl.field_name] = val
		}
	} else {
		h[fl.field_name] = fl.field_value
	}
	return parsed + len(CRLF), false, nil
}

func (h Headers) Get(key string) (value string, ok bool) {
	key = strings.ToLower(key)
	value, ok = h[key]
	return value, ok
}

func (h Headers) Set(key string, value string) {
	key = strings.ToLower(key)
	h[key] = value
}

func parseHeader(data []byte) (field *field_line, n int, err error) {
	var buffer bytes.Buffer
	parsed, err := buffer.Write(data)
	if err != nil {
		return nil, 0, err
	}

	field, err = parseHeaderString(buffer.String())
	if err != nil {
		return nil, 0, err
	}
	return field, parsed, nil
}

func parseHeaderString(line string) (field *field_line, err error) {
	idx := strings.Index(line, SEPARATOR)
	if idx != -1 {
		if len(strings.Split(strings.Trim(line[:idx], " "), " ")) != 1 {
			return nil, fmt.Errorf("invalid format, expected \"field-name: field-value\", got=%s", line)
		}
	}
	if idx == -1 {
		return nil, fmt.Errorf("invalid format, header must contain \":\", got=%s", line)
	}
	if idx == 0 {
		return nil, fmt.Errorf("invalid format, header must have a key for each value. got=%s", line)
	}
	if line[idx-1] == ' ' {
		return nil, fmt.Errorf("illegal trailing space before \":\" at idx=%d in %s", idx-1, line)
	}
	parts := strings.SplitN(line, SEPARATOR, 2)

	err = isValidToken(strings.TrimLeft(parts[0], " "))
	if err != nil {
		return nil, err
	}

	field = &field_line{
		field_name:  strings.ToLower(strings.TrimLeft(parts[0], " ")),
		field_value: strings.ToLower(strings.Trim(parts[1], " ")),
	}

	return field, nil
}

func isValidToken(t string) error {
	delimeter := []rune{'!', '#', '$', '%', '&', '\'', '*', '+', '-', '.', '^', '_', '`', '|', '~'}
	for _, char := range t {
		if (char < 'a' || char > 'z') && (char < 'A' || char > 'Z') && (char < '0' || char > '9') && !(slices.Contains(delimeter, char)) {
			return fmt.Errorf("invalid field name at %d, got=%s", char, t)
		}
	}
	return nil
}
