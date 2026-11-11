package middleware

import (
	"bufio"
	"io"
	"strconv"
	"testing"
	"time"

	"azugo.io/azugo"

	"azugo.io/core/http"
	"github.com/go-quicktest/qt"
	"github.com/valyala/fasthttp"
)

// uploadSize exceeds what the server reads before calling the handler.
const uploadSize = 1 << 20

func readBody(ctx *azugo.Context) {
	b, err := io.ReadAll(ctx.Request().BodyStream())
	if err != nil {
		ctx.StatusCode(http.StatusRequestTimeout)

		return
	}

	ctx.Text(strconv.Itoa(len(b)))
}

// slowUpload sends a PUT whose body pauses for delay after its first part and returns the response.
func slowUpload(t *testing.T, a *azugo.TestApp, path string, delay time.Duration) *fasthttp.Response {
	t.Helper()

	conn, err := a.TestClient().Client().Dial("test")
	qt.Assert(t, qt.IsNil(err))

	defer conn.Close()

	_, err = conn.Write([]byte("PUT " + path + " HTTP/1.1\r\nHost: test\r\nContent-Length: " + strconv.Itoa(uploadSize) + "\r\n\r\n"))
	qt.Assert(t, qt.IsNil(err))

	_, err = conn.Write(make([]byte, 64<<10))
	qt.Assert(t, qt.IsNil(err))

	time.Sleep(delay)

	// The server may already have answered and closed the connection.
	_, _ = conn.Write(make([]byte, uploadSize-64<<10))

	resp := fasthttp.AcquireResponse()
	qt.Assert(t, qt.IsNil(resp.Read(bufio.NewReader(conn))))

	return resp
}

// slowDownload sends a GET, waits delay before reading and returns the response.
func slowDownload(t *testing.T, a *azugo.TestApp, path string, delay time.Duration) (*fasthttp.Response, error) {
	t.Helper()

	conn, err := a.TestClient().Client().Dial("test")
	qt.Assert(t, qt.IsNil(err))

	defer conn.Close()

	_, err = conn.Write([]byte("GET " + path + " HTTP/1.1\r\nHost: test\r\n\r\n"))
	qt.Assert(t, qt.IsNil(err))

	time.Sleep(delay)

	resp := fasthttp.AcquireResponse()

	return resp, resp.Read(bufio.NewReader(conn))
}

// streamBody writes the response in many flushed chunks, so a client that does not read blocks it.
func streamBody(ctx *azugo.Context) {
	ctx.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
		chunk := make([]byte, 4<<10)

		for range 64 {
			if _, err := w.Write(chunk); err != nil {
				return
			}

			if err := w.Flush(); err != nil {
				return
			}
		}
	})
}

func TestReadTimeout(t *testing.T) {
	a := azugo.NewTestApp()

	short := a.Group("/short")
	short.Use(ReadTimeout(20 * time.Millisecond))
	short.Put("/upload", readBody)

	long := a.Group("/long")
	long.Use(ReadTimeout(time.Second))
	long.Put("/upload", readBody)

	a.Start(t)
	defer a.Stop()

	resp := slowUpload(t, a, "/short/upload", 200*time.Millisecond)
	defer fasthttp.ReleaseResponse(resp)

	qt.Check(t, qt.Equals(resp.StatusCode(), http.StatusRequestTimeout))

	resp = slowUpload(t, a, "/long/upload", 100*time.Millisecond)
	defer fasthttp.ReleaseResponse(resp)

	qt.Check(t, qt.Equals(resp.StatusCode(), http.StatusOK))
	qt.Check(t, qt.Equals(string(resp.Body()), strconv.Itoa(uploadSize)))
}

func TestWriteTimeout(t *testing.T) {
	a := azugo.NewTestApp()

	short := a.Group("/short")
	short.Use(WriteTimeout(20 * time.Millisecond))
	short.Get("/download", streamBody)

	long := a.Group("/long")
	long.Use(WriteTimeout(time.Second))
	long.Get("/download", streamBody)

	a.Start(t)
	defer a.Stop()

	resp, err := slowDownload(t, a, "/short/download", 200*time.Millisecond)
	defer fasthttp.ReleaseResponse(resp)

	qt.Check(t, qt.IsNotNil(err))

	resp, err = slowDownload(t, a, "/long/download", 100*time.Millisecond)
	defer fasthttp.ReleaseResponse(resp)

	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(resp.StatusCode(), http.StatusOK))
	qt.Check(t, qt.HasLen(resp.Body(), 64*4<<10))
}
