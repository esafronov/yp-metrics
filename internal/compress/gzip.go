package compress

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"strings"
)

// gzipWriter реализует интерфейс http.ResponseWriter и позволяет прозрачно для сервера
// сжимать передаваемые данные и выставлять правильные HTTP-заголовки
type gzipWriter struct {
	w        http.ResponseWriter
	zw       *gzip.Writer
	compress bool //flag if we need to compress response content
}

func newGzipWriter(w http.ResponseWriter) *gzipWriter {
	return &gzipWriter{
		w:        w,
		zw:       gzip.NewWriter(w),
		compress: false,
	}
}

func (c *gzipWriter) Header() http.Header {
	return c.w.Header()
}

func (c *gzipWriter) Write(p []byte) (int, error) {
	//if we need to gzip content
	if c.compress {
		return c.zw.Write(p)
	}
	return c.w.Write(p)
}

func (c *gzipWriter) WriteHeader(statusCode int) {
	contentType := c.w.Header().Get("Content-Type")
	//if we need to compress
	if contentType == "application/json" || contentType == "text/html" {
		//if http code is not a error
		if statusCode < 300 {
			c.compress = true
			c.w.Header().Set("Content-Encoding", "gzip")
		}
	}
	c.w.WriteHeader(statusCode)
}

// Close закрывает gzip.Writer и досылает все данные из буфера.
func (c *gzipWriter) Close() error {
	//close only if we use gzip
	if c.compress {
		return c.zw.Close()
	}
	return nil
}

// GZipReader реализует интерфейс io.ReadCloser и позволяет прозрачно для сервера
// декомпрессировать получаемые от клиента данные
type gzipReader struct {
	r  io.ReadCloser
	zr *gzip.Reader
}

func newGzipReader(r io.ReadCloser) (*gzipReader, error) {
	zr, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}
	return &gzipReader{
		r:  r,
		zr: zr,
	}, nil
}

func (c gzipReader) Read(p []byte) (n int, err error) {
	return c.zr.Read(p)
}

func (c *gzipReader) Close() error {
	if err := c.r.Close(); err != nil {
		return err
	}
	return c.zr.Close()
}

// server middleware for compressing data
func GzipCompressing(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// по умолчанию устанавливаем оригинальный http.ResponseWriter как тот,
		// который будем передавать следующей функции
		ow := w

		// проверяем, что клиент умеет получать от сервера сжатые данные в формате gzip
		acceptEncoding := r.Header.Get("Accept-Encoding")
		supportsGzip := strings.Contains(acceptEncoding, "gzip")

		if supportsGzip {
			// оборачиваем оригинальный http.ResponseWriter новым с поддержкой сжатия
			cw := newGzipWriter(w)
			// меняем оригинальный http.ResponseWriter на новый
			ow = cw
			// не забываем отправить клиенту все сжатые данные после завершения middleware
			defer cw.Close()
		}

		// проверяем, что клиент отправил серверу сжатые данные в формате gzip
		contentEncoding := r.Header.Get("Content-Encoding")
		sendsGzip := strings.Contains(contentEncoding, "gzip")
		if sendsGzip {
			// оборачиваем тело запроса в io.Reader с поддержкой декомпрессии
			cr, err := newGzipReader(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			// меняем тело запроса на новое
			r.Body = cr
			defer cr.Close()
		}

		// передаём управление хендлеру
		h.ServeHTTP(ow, r)
	})
}

// compress bytes to buffer
func GzipToBuffer(data []byte, buf *bytes.Buffer) error {
	wr := gzip.NewWriter(buf)
	_, err := wr.Write(data)
	if err != nil {
		return err
	}
	err = wr.Close()
	if err != nil {
		return err
	}
	return nil
}
