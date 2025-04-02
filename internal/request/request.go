package request

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"

	"github.com/Barrioslopezfd/httpfromtcp/internal/headers"
)

type State int

const (
	INITIALIZED State = iota
	PARSING_HEADERS
	PARSING_BODY
	DONE
)

const CRLF = "\r\n"

type Request struct {
	RequestLine RequestLine
	ParserState State
	Headers     headers.Headers
	Body        []byte
}

type RequestLine struct {
	HttpVersion   string
	RequestTarget string
	Method        string
}

func RequestFromReader(reader io.Reader) (*Request, error) {
	req := &Request{
		ParserState: INITIALIZED,
		Headers:     make(map[string]string),
	}

	buffer := make([]byte, 8)
	var readBuffer bytes.Buffer
	bytesRead := 0
	bytesParsed := 0
	for {
		n, err := reader.Read(buffer)
		if err != nil {
			return nil, err
		}
		bytesRead += n
		_, err = readBuffer.Write(buffer[:n])
		if err != nil {
			return nil, err
		}
		consumed, err := req.parse(readBuffer.Bytes())
		if err != nil {
			return nil, err
		}
		if consumed == 0 {
			buffer = make([]byte, bytesRead)
			continue
		}
		bytesParsed += consumed
		if req.ParserState == DONE {
			break
		}
	}

	return req, nil
}

func (r *Request) parse(data []byte) (int, error) {
	if r.ParserState == DONE {
		return 0, errors.New("\"Parse State\"=done")
	}
	reqLine, consumed, err := parseRequestLine(data)
	if err != nil {
		return 0, err
	}
	if consumed == 0 {
		return 0, nil
	}
	r.RequestLine = *reqLine
	r.ParserState = PARSING_HEADERS
	for r.ParserState == PARSING_HEADERS {
		parsed, done, err := r.Headers.Parse(data[consumed:])
		if err != nil {
			return 0, err
		}
		consumed += parsed
		if parsed == 0 {
			r.ParserState = INITIALIZED
		}
		if done {
			r.ParserState = PARSING_BODY
		}
	}

	for r.ParserState == PARSING_BODY {
		parsed, done, err := r.parseBody(data[consumed:])
		if err != nil {
			return 0, err
		}
		consumed += parsed
		if parsed == 0 {
			r.ParserState = INITIALIZED
		}
		if done {
			r.ParserState = DONE
		}
	}

	return consumed, nil
}

func (r *Request) parseBody(body []byte) (int, bool, error) {
	value, ok := r.Headers.Get("content-length")
	if !ok {
		return 0, true, nil
	}
	r.Body = body
	val, err := strconv.Atoi(value)
	if err != nil {
		return 0, false, fmt.Errorf("invalid content-length, got=%s", value)
	}
	if len(r.Body) > val {
		return 0, false, fmt.Errorf("body is too long, expected=%d, got=%d", val, len(r.Body))
	}
	if len(r.Body) < val {
		return 0, false, nil
	}
	return val, true, nil
}

func parseRequestLine(b []byte) (*RequestLine, int, error) {
	idx := bytes.Index(b, []byte(CRLF))
	if idx == -1 {
		return nil, 0, nil
	}

	var buffer bytes.Buffer
	parsed, err := buffer.Write(b[:idx])
	if err != nil {
		return nil, 0, err
	}
	reqLineStr := buffer.String()
	reqLine, err := parseRequestLineString(reqLineStr)
	if err != nil {
		return nil, 0, err
	}

	return reqLine, parsed + 2, nil
}

func parseRequestLineString(requestLine string) (*RequestLine, error) {
	reqLineSlice := strings.Split(requestLine, " ")
	if len(reqLineSlice) != 3 {
		return nil, fmt.Errorf("request line must contain 3 parts, got=%d. string=%s", len(reqLineSlice), reqLineSlice)
	}

	method := reqLineSlice[0]
	target := reqLineSlice[1]
	httpVer := reqLineSlice[2]

	for _, char := range method {
		if char < 'A' || char > 'Z' {
			return nil, errors.New("invalid method")
		}
	}

	if len(target) < 1 {
		return nil, fmt.Errorf("path must contain at least 1 character, got=%d", len(target))
	}

	if target[0] != '/' {
		return nil, fmt.Errorf("path must start with '/', got=%c", target[0])
	}

	if len(target) > 1 {
		err := isValidToken(target[1:])
		if err != nil {
			return nil, err
		}
	}

	if httpVer != "HTTP/1.1" {
		return nil, errors.New("invalid http version")
	}
	version := strings.Split(httpVer, "/")[1]

	return &RequestLine{
		HttpVersion:   version,
		RequestTarget: target,
		Method:        method,
	}, nil
}

func isValidToken(t string) error {
	specialChars := []rune{'/', '.', '-'}
	for _, char := range t {
		if (char < '0' || char > '9') && (char < 'a' || char > 'z') && !(slices.Contains(specialChars, char)) {
			return fmt.Errorf("invalid path at %d, got=%s", char, t)
		}
	}
	return nil
}
