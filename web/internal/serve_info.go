package internal

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

type serveInfo struct {
	statusCode     int
	contentLength  int64
	contentWritten int64
	expires        *time.Time

	errorType             errorType
	error                 error
	responseWriter        http.ResponseWriter
	responseHeader        http.Header
	responseWrittenHeader bool
}

func NewServeInfo(w http.ResponseWriter) ServeInfo {
	s := &serveInfo{
		responseWriter: w,
		responseHeader: make(http.Header),
	}
	return s
}

func (si *serveInfo) GetStatusCode() int {
	return si.statusCode
}

func (si *serveInfo) GetContentInfo() (int64, int64) {
	return si.contentLength, si.contentWritten
}

func (si *serveInfo) GetExpires() *time.Time {
	if si.expires == nil {
		return nil
	}

	e := *si.expires
	return &e
}

func (si *serveInfo) HasError() bool {
	return si.error != nil
}

func (si *serveInfo) GetError() (int, error) {
	return int(si.errorType), si.error
}

func (si *serveInfo) OnCacheNotFound(err error) ServeInfo {
	si.statusCode = http.StatusNotFound
	si.errorType = ErrorCacheNotFound
	si.error = err

	return si
}

func (si *serveInfo) OnNoStatusCode(t errorType, format string, a ...interface{}) ServeInfo {
	si.statusCode = http.StatusNotImplemented
	si.errorType = t
	si.error = fmt.Errorf(format, a...)

	return si
}

func (si *serveInfo) OnBrokenHeader(t errorType, format string, a ...interface{}) ServeInfo {
	si.statusCode = http.StatusInternalServerError
	si.errorType = t
	si.error = fmt.Errorf(format, a...)

	return si
}

func (si *serveInfo) SetStatusCode(statusCode int) {
	si.statusCode = statusCode
	si.errorType = 0
	si.error = nil
}

func (si *serveInfo) SetExpires(e time.Time) {
	si.expires = &e
}

func (si *serveInfo) SetContentLength(value int64) {
	si.contentLength = value
	si.responseHeader.Set("Content-Length", fmt.Sprintf("%d", value))
}

func (si *serveInfo) AddHeader(key string, value string) {
	si.responseHeader.Add(key, value)
}

func (si *serveInfo) WriteBody(bytes []byte) {
	if bytes != nil {
		si.SetContentLength(int64(len(bytes)))
		si.writeHeader()

		written, err := si.responseWriter.Write(bytes)
		si.contentWritten = int64(written)

		if err != nil {
			si.errorType = ErrorWriteBody
			si.error = err
		}
	}
}

func (si *serveInfo) CopyBody(source io.Reader) {
	if si.contentLength == 0 {
		return
	}

	si.writeHeader()

	written, err := io.CopyN(si.responseWriter, source, si.contentLength)
	si.contentWritten = written

	if err != nil {
		si.errorType = ErrorCopyBody
		si.error = err
	}
}

func (si *serveInfo) Flush() ServeInfo {
	si.writeHeader()

	return si
}

func (si *serveInfo) writeHeader() {
	if !si.responseWrittenHeader {
		si.responseWrittenHeader = true

		responseWriterHeader := si.responseWriter.Header()
		for key, values := range si.responseHeader {
			for _, value := range values {
				responseWriterHeader.Add(key, value)
			}
		}

		si.responseWriter.WriteHeader(si.statusCode)
	}
}
