package main

import (
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/Barrioslopezfd/httpfromtcp/cmd/server"
	"github.com/Barrioslopezfd/httpfromtcp/internal/headers"
	"github.com/Barrioslopezfd/httpfromtcp/internal/request"
	"github.com/Barrioslopezfd/httpfromtcp/internal/response"
)

const port = 42069

func main() {
	server, err := server.Serve(handler, port)
	if err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
	defer server.Close()
	log.Println("Server started on port", port)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	log.Println("Server gracefully stopped")
}

func handler(w *response.Writer, r *request.Request) {
	if strings.HasPrefix(r.RequestLine.RequestTarget, "/httpbin") {
		handlerChunk(w, r)
		return
	}
	if r.RequestLine.RequestTarget == "/yourproblem" {
		handler400(w, r)
		return
	}

	if r.RequestLine.RequestTarget == "/myproblem" {
		handler500(w, r)
		return
	}
	handler200(w, r)
}

func handler200(w *response.Writer, r *request.Request) {
	w.WriteStatusLine(response.OK)
	body := toHtmlString(200, "Success!!", "Your request was an absolute banger.")
	r.Headers.Replace("Content-Type", "text/html")
	r.Headers.Replace("Content-Length", fmt.Sprint(len(body)))
	w.WriteHeaders(r.Headers)
	w.WriteBody(fmt.Appendf(nil, body))
}

func handler500(w *response.Writer, r *request.Request) {
	w.WriteStatusLine(response.INTERNAL_SERVER_ERROR)
	body := toHtmlString(500, "Internal Server Error", "Okay, you know what? This one is on me")
	r.Headers.Replace("Content-Type", "text/html")
	r.Headers.Replace("Content-Length", fmt.Sprint(len(body)))
	w.WriteHeaders(r.Headers)
	w.WriteBody(fmt.Appendf(nil, body))
}

func handler400(w *response.Writer, r *request.Request) {
	body := toHtmlString(400, "Bad Request", "Your request honestly kinda sucked.")
	w.WriteStatusLine(response.BAD_REQUEST)
	r.Headers.Replace("Content-Type", "text/html")
	r.Headers.Replace("Content-Length", fmt.Sprint(len(body)))
	w.WriteHeaders(r.Headers)
	w.WriteBody(fmt.Appendf(nil, body))
}

func toHtmlString(code int, errorMsg string, body string) string {
	return fmt.Sprintf(`<html>
	<head>
	<title>%d %s</title>
	</head>
	<body>
	<h1>%s</h1>
	<p>%s</p>
	</body>
	</html>`, code, errorMsg, errorMsg, body)
}

func handlerChunk(w *response.Writer, r *request.Request) {
	path := strings.TrimPrefix(r.RequestLine.RequestTarget, "/httpbin/")
	url := "https://httpbin.org/" + path
	res, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	defer res.Body.Close()
	err = w.WriteStatusLine(response.OK)
	if err != nil {
		log.Fatal(err)
	}
	h := response.GetDefaultHeaders()
	h.Remove("Content-Length")
	h.Replace("Content-Type", "text/html")
	h.Replace("Transfer-Encoding", "chunked")
	h.Set("trailer", "x-content-sha256, x-content-length")
	w.WriteHeaders(h)
	buffer := make([]byte, 1024)
	body := make([]byte, 0)
	for {
		n, err := res.Body.Read(buffer)
		fmt.Println("Read", n, "bytes")
		if n > 0 {
			_, err := w.WriteChunkedBody(buffer[:n])
			if err != nil {
				fmt.Println("Error writing chunked body:", err.Error())
				break
			}
			body = append(body, buffer[:n]...)
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Println("Error reading response body:", err.Error())
			break
		}
	}
	_, err = w.WriteChunkedBodyDone()
	if err != nil {
		fmt.Println("Error writing chunked body done:", err.Error())
	}

	trailer := headers.NewHeaders()
	sha := sha256.Sum256(body)
	trailer.Set("X-Content-SHA256", fmt.Sprintf("%x", sha))
	trailer.Set("X-Content-Length", fmt.Sprint(len(body)))
	err = w.WriteTrailers(trailer)
	if err != nil {
		fmt.Println(err.Error())
	}
}
