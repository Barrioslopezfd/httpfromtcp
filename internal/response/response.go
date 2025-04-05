package response

import (
	"fmt"
	"io"

	"github.com/Barrioslopezfd/httpfromtcp/internal/headers"
)

type Code int

const (
	OK                    Code = 200
	BAD_REQUEST           Code = 400
	INTERNAL_SERVER_ERROR Code = 500
)

var statusCode = map[Code]string{
	OK:                    "HTTP/1.1 200 OK",
	BAD_REQUEST:           "HTTP/1.1 400 Bad Request",
	INTERNAL_SERVER_ERROR: "HTTP/1.1 500 Internal Server Error",
}

type state int

const (
	STATUS_LINE state = iota
	HEADERS
	BODY
)

type Writer struct {
	Writer      io.Writer
	writerState state
}

func (w *Writer) WriteStatusLine(code Code) error {
	if w.writerState != STATUS_LINE {
		return fmt.Errorf("error, you have to start from the status line")
	}
	msg, ok := statusCode[code]
	if !ok {
		msg = fmt.Sprintf("HTTP/1.1 %d ", code)
	}

	msg += "\r\n"
	if _, err := w.Writer.Write([]byte(msg)); err != nil {
		return err
	}

	w.writerState = HEADERS

	return nil
}

func GetDefaultHeaders() headers.Headers {
	h := headers.NewHeaders()

	h.Set("Content-Length", "0")
	h.Set("Connection", "close")
	h.Set("Content-Type", "text/plain")
	return h
}

func (w *Writer) WriteHeaders(headers headers.Headers) error {
	if w.writerState != HEADERS {
		return fmt.Errorf("trying to write to header without write header state")
	}
	for key, value := range headers {
		_, err := w.Writer.Write(fmt.Appendf(nil, "%s: %s\r\n", key, value))
		if err != nil {
			return fmt.Errorf("error while writing headers on loop, err=%s", err.Error())
		}
	}
	_, err := w.Writer.Write([]byte("\r\n"))
	if err != nil {
		return fmt.Errorf("error while writing headers, err=%s", err.Error())
	}
	w.writerState = BODY
	return nil
}

func (w *Writer) WriteBody(body []byte) (int, error) {
	if w.writerState != BODY {
		return 0, fmt.Errorf("error, headers not found")
	}
	n, err := w.Writer.Write(body)
	if err != nil {
		return 0, fmt.Errorf("error writing body, err=%s", err.Error())
	}
	return n, nil
}

func (w *Writer) WriteChunkedBody(body []byte) (int, error) {
	if w.writerState != BODY {
		return 0, fmt.Errorf("error, headers not found")
	}
	total := 0
	lengthLine := fmt.Sprintf("%x\r\n", len(body))
	read, err := w.Writer.Write(fmt.Appendf(nil, lengthLine))
	if err != nil {
		return total, err
	}
	total += read
	read, err = w.Writer.Write(body)
	if err != nil {
		return total, err
	}
	total += read
	read, err = w.Writer.Write([]byte("\r\n"))
	if err != nil {
		return total, err
	}
	total += read
	return total, nil
}

func (w *Writer) WriteChunkedBodyDone() (int, error) {
	if w.writerState != BODY {
		return 0, fmt.Errorf("error, headers not found")
	}
	return w.Writer.Write([]byte("0\r\n"))
}

func (w *Writer) WriteTrailers(h headers.Headers) error {
	buffer := ""
	for key, value := range h {
		buffer += fmt.Sprintf("%s: %s\r\n", key, value)
	}
	buffer += "\r\n"
	_, err := w.Writer.Write([]byte(buffer))
	return err
}
