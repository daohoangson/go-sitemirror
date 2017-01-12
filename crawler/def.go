package crawler

import (
	"bytes"
	"net/http"
	"net/url"
)

// Crawler must be created with New
type Crawler interface {
	init(*http.Client)

	SetWorkerCount(int) error
	GetWorkerCount() int
	SetOnDownloaded(downloadedFunc)
	SetOnURLShouldQueue(urlShouldQueueFunc)
	IsWorkersRunning() bool
	GetQueuedCount() int
	GetDownloadedCount() int
	GetLinkFoundCount() int
	Start()
	Download(string)
	Next() *Downloaded
	NextOrNil() *Downloaded
}

// Downloaded struct contains parsed data after downloading an url.
type Downloaded struct {
	BaseURL        *url.URL
	BodyString     string
	BodyBytes      []byte
	ContentType    string
	Error          error
	HeaderLocation *url.URL
	Links          []Link
	StatusCode     int

	buffer *bytes.Buffer
}

// Link struct is an extracted link from download result.
type Link struct {
	Context urlContext
	Offset  int
	Length  int
	URL     *url.URL
}

const (
	// CSSUri url from url()
	CSSUri urlContext = 1 + iota
	// HTMLTagA url from <a href=""></a>
	HTMLTagA
	// HTMLTagImg url from <img src="" />
	HTMLTagImg
	// HTMLTagLinkStylesheet url from <link rel="stylesheet" href="" />
	HTMLTagLinkStylesheet
	// HTMLTagScript url from <script src="" />
	HTMLTagScript
)

type urlContext int
