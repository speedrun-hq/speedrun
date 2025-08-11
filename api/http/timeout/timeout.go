package timeout

import (
	"bytes"
	"fmt"
	"net/http"
	"runtime/debug"
	"sync"
	"time"

	"github.com/gin-contrib/timeout"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

// panicChan transmits both the panic value and the stack trace.
type panicInfo struct {
	Value any
	Stack []byte
}

// New wraps a handler and aborts the process of the handler if the timeout is reached
// This code is copied from https://github.com/gin-contrib/timeout and modified
// to support Zerolog logging
func New(reqTimeout time.Duration, log zerolog.Logger) gin.HandlerFunc {
	bufPool := &timeout.BufferPool{}

	return func(c *gin.Context) {
		// Swap the response writer with a buffered writer.
		var (
			w      = c.Writer
			buffer = bufPool.Get()
			tw     = NewWriter(w, buffer)
		)

		c.Writer = tw
		buffer.Reset()

		// Create a copy of the context before starting the goroutine to avoid data race
		cCopy := c.Copy()

		// Set the copied context's writer to our timeout writer to ensure proper buffering
		cCopy.Writer = tw

		// Channel to signal handler completion.
		finish := make(chan struct{}, 1)
		panicChan := make(chan panicInfo, 1)

		// Run the handler in a separate goroutine to enforce timeout and catch panics.
		go func() {
			defer func() {
				if p := recover(); p != nil {
					// Capture both the panic value and the stack trace.
					panicChan <- panicInfo{
						Value: p,
						Stack: debug.Stack(),
					}
				}
			}()

			// Use the copied context to avoid data race when running handler in a goroutine.
			c.Next()

			finish <- struct{}{}
		}()

		select {
		case pi := <-panicChan:
			// Handler panicked: free buffer, restore writer, and print stack trace if in debug mode.
			tw.FreeBuffer()
			c.Writer = w

			log.Error().
				Any("panic", pi.Value).
				Str("middleware", "timeout").
				Str("http.method", c.Request.Method).
				Str("http.path", c.Request.URL.Path).
				Msg("HTTP request panicked")

		case <-finish:
			// Handler finished successfully: flush buffer to response.
			tw.mu.Lock()
			defer tw.mu.Unlock()
			dst := tw.ResponseWriter.Header()
			for k, vv := range tw.Header() {
				dst[k] = vv
			}

			// Write the status code if it was set, otherwise use 200
			if tw.code != 0 {
				tw.ResponseWriter.WriteHeader(tw.code)
			}

			// Only write content if there's any
			if buffer.Len() > 0 {
				if _, err := tw.ResponseWriter.Write(buffer.Bytes()); err != nil {
					panic(err)
				}
			}

			tw.FreeBuffer()
			bufPool.Put(buffer)

		case <-time.After(reqTimeout):
			tw.mu.Lock()

			// Handler timed out: set timeout flag and clean up
			tw.timeout = true
			tw.FreeBuffer()
			bufPool.Put(buffer)
			tw.mu.Unlock()

			// Create a fresh context for the timeout response
			// Important: check if headers were already written
			timeoutCtx := c.Copy()
			timeoutCtx.Writer = w

			// Only write timeout response if headers haven't been written to original writer
			if !w.Written() {
				defaultResponse(timeoutCtx)
			}

			// Abort the context to prevent further middleware execution after timeout
			c.AbortWithStatus(http.StatusRequestTimeout)

			log.Warn().
				Str("http.method", c.Request.Method).
				Str("http.path", c.Request.URL.Path).
				Msg("HTTP request timed out")
		}
	}
}

// Writer is a writer with memory buffer
type Writer struct {
	gin.ResponseWriter
	body         *bytes.Buffer
	headers      http.Header
	mu           sync.Mutex
	timeout      bool
	wroteHeaders bool
	code         int
	size         int
}

// NewWriter will return a timeout.Writer pointer
func NewWriter(w gin.ResponseWriter, buf *bytes.Buffer) *Writer {
	return &Writer{ResponseWriter: w, body: buf, headers: make(http.Header)}
}

// WriteHeaderNow the reason why we override this func is:
// once calling the func WriteHeaderNow() of based gin.ResponseWriter,
// this Writer can no longer apply the cached headers to the based
// gin.ResponseWriter. see test case `TestWriter_WriteHeaderNow` for details.
func (w *Writer) WriteHeaderNow() {
	if !w.wroteHeaders {
		if w.code == 0 {
			w.code = http.StatusOK
		}

		// Copy headers from our cache to the underlying ResponseWriter
		dst := w.ResponseWriter.Header()
		for k, vv := range w.headers {
			dst[k] = vv
		}

		w.WriteHeader(w.code)
	}
}

// WriteHeader sends an HTTP response header with the provided status code.
// If the response writer has already written headers or if a timeout has occurred,
// this method does nothing.
func (w *Writer) WriteHeader(code int) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.timeout || w.wroteHeaders {
		return
	}

	// gin is using -1 to skip writing the status code
	// see https://github.com/gin-gonic/gin/blob/a0acf1df2814fcd828cb2d7128f2f4e2136d3fac/response_writer.go#L61
	if code == -1 {
		return
	}

	checkWriteHeaderCode(code)

	// Copy headers from our cache to the underlying ResponseWriter
	dst := w.ResponseWriter.Header()
	for k, vv := range w.headers {
		dst[k] = vv
	}

	w.writeHeader(code)
	w.ResponseWriter.WriteHeader(code)
}

func (w *Writer) writeHeader(code int) {
	w.wroteHeaders = true
	w.code = code
}

// Header will get response headers
func (w *Writer) Header() http.Header {
	return w.headers
}

// Write will write data to response body
func (w *Writer) Write(data []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.timeout || w.body == nil {
		return 0, nil
	}

	n, err := w.body.Write(data)
	w.size += n

	return n, err
}

// WriteString will write string to response body
func (w *Writer) WriteString(s string) (int, error) {
	n, err := w.Write([]byte(s))
	w.size += n
	return n, err
}

func (w *Writer) Size() int {
	return w.size
}

// FreeBuffer will release buffer pointer
func (w *Writer) FreeBuffer() {
	// if not reset body,old bytes will put in bufPool
	w.body.Reset()
	w.size = -1
	w.body = nil
}

// Status we must override Status func here,
// or the http status code returned by gin.Context.Writer.Status()
// will always be 200 in other custom gin middlewares.
func (w *Writer) Status() int {
	if w.code == 0 || w.timeout {
		return w.ResponseWriter.Status()
	}
	return w.code
}

func checkWriteHeaderCode(code int) {
	if code < 100 || code > 999 {
		panic(fmt.Sprintf("invalid http status code: %d", code))
	}
}

func defaultResponse(c *gin.Context) {
	c.String(http.StatusRequestTimeout, http.StatusText(http.StatusRequestTimeout))
}
