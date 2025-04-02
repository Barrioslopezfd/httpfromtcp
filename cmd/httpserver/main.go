package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/Barrioslopezfd/httpfromtcp/cmd/server"
	"github.com/Barrioslopezfd/httpfromtcp/internal/request"
	"github.com/Barrioslopezfd/httpfromtcp/internal/response"
)

const port = 42069

func main() {
	server, err := server.Serve(handlerChunk, port)
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

func handler(w *response.Writer, req *request.Request) {
	if req.RequestLine.RequestTarget == "/yourproblem" {
		body := toHtmlString(400, "Bad Request", "Your request honestly kinda sucked.")
		w.WriteStatusLine(response.BAD_REQUEST)
		req.Headers.Set("Content-Type", "text/html")
		req.Headers.Set("Content-Length", fmt.Sprint(len(body)))
		w.WriteHeaders(req.Headers)
		w.WriteBody(fmt.Appendf(nil, body))
		return
	}

	if req.RequestLine.RequestTarget == "/myproblem" {
		w.WriteStatusLine(response.INTERNAL_SERVER_ERROR)
		body := toHtmlString(500, "Internal Server Error", "Okay, you know what? This one is on me")
		req.Headers.Set("Content-Type", "text/html")
		req.Headers.Set("Content-Length", fmt.Sprint(len(body)))
		w.WriteHeaders(req.Headers)
		w.WriteBody(fmt.Appendf(nil, body))
		return
	}
	w.WriteStatusLine(response.OK)
	body := toHtmlString(200, "Success!!", "Your request was an absolute banger.")
	req.Headers.Set("Content-Type", "text/html")
	req.Headers.Set("Content-Length", fmt.Sprint(len(body)))
	w.WriteHeaders(req.Headers)
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

func handlerChunk(w *response.Writer, req *request.Request) {
	if strings.HasPrefix(req.RequestLine.RequestTarget, "/httpbin/") {
		httpbinPath := strings.TrimPrefix(req.RequestLine.RequestTarget, "/httpbin/")

		httpbinURL := "https://httpbin.org/" + httpbinPath

		httpbinResp, err := http.Get(httpbinURL)
		if err != nil {
			log.Fatal(err)
		}
		defer httpbinResp.Body.Close()

		w.WriteStatusLine(response.Code((httpbinResp.StatusCode)))

		req.Headers.Set("Transfer-Encoding", "chunked")
		for key, value := range httpbinResp.Header {
			req.Headers.Set(key, strings.Join(value, ","))
		}

		w.WriteHeaders(req.Headers)

		buffer := make([]byte, 1024)
		for {
			n, err := httpbinResp.Body.Read(buffer)
			if n > 0 {
				w.WriteChunkedBody(buffer[:n])
			}
			if err != nil {
				break
			}
		}

		w.WriteChunkedBodyDone()
		return
	}
}
