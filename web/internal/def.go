package internal

import (
	"io"
	"time"
)

type ServeInfo interface {
	GetStatusCode() int
	GetContentInfo() (int64, int64)
	GetExpires() *time.Time
	HasError() bool
	GetError() (int, error)

	OnCacheNotFound(error) ServeInfo
	OnNoStatusCode(errorType, string, ...interface{}) ServeInfo
	OnBrokenHeader(errorType, string, ...interface{}) ServeInfo

	SetStatusCode(int)
	SetExpires(time.Time)
	SetContentLength(int64)
	AddHeader(string, string)
	WriteBody(bytes []byte)
	CopyBody(source io.Reader)

	Flush() ServeInfo
}

const (
	ErrorCacheNotFound errorType = 1 + iota
	ErrorReadLine
	ErrorParseLine
	ErrorParseInt
	ErrorWriteBody
	ErrorCopyBody
	ErrorOther
)

type errorType int
