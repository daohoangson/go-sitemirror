package internal

import (
	"io"
	"time"
)

// ServeInfo represents an ongoing user request
type ServeInfo interface {
	GetStatusCode() int
	GetContentInfo() (int64, int64)
	GetExpires() *time.Time
	HasError() bool
	GetError() (int, error)

	OnMethodNotAllowed() ServeInfo
	OnCacheNotFound(error) ServeInfo
	OnNoStatusCode(errorType, string, ...interface{}) ServeInfo
	OnBrokenHeader(errorType, string, ...interface{}) ServeInfo
	OnCrossHostInvalidPath() ServeInfo
	OnCrossHostRef() ServeInfo

	SetStatusCode(int)
	SetExpires(time.Time)
	SetContentLength(int64)
	AddHeader(string, string)
	WriteBody([]byte)
	CopyBody(source io.Reader)

	Flush() ServeInfo
}

const (
	// ErrorCacheNotFound serve info error type when existing cache cannot be found
	ErrorCacheNotFound errorType = 1 + iota
	// ErrorReadLine serve info error type when line cannot be read as expected
	ErrorReadLine
	// ErrorParseLine serve info error type when line cannot be parsed as expected
	ErrorParseLine
	// ErrorParseInt serve info error type when value cannot be parsed as integer
	ErrorParseInt
	// ErrorWriteBody serve info error type when occur an error during body write
	ErrorWriteBody
	// ErrorCopyBody serve info error type when occur an error during body copy
	ErrorCopyBody
	// ErrorCrossHostRefOnNonCrossHost serve info error type when a cross-host reference is found in non cross-host context
	ErrorCrossHostRefOnNonCrossHost
)

type errorType int
