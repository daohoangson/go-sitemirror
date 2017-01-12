package crawler

import (
	"bytes"
	"net/http"
	"net/url"
)

// Crawler must be created with New
type Crawler interface {
	init(*http.Client)

	SetAutoDownloadDepth(int)
	GetAutoDownloadDepth() int
	SetWorkerCount(int) error
	GetWorkerCount() int

	SetOnURLShouldQueue(func(*url.URL) bool)
	SetOnDownload(func(*url.URL))
	SetOnDownloaded(func(*Downloaded))

	IsWorkersRunning() bool
	GetQueuedCount() int
	GetDownloadedCount() int
	GetLinkFoundCount() int

	Start()
	Download(*url.URL)
	DownloadURL(string) error
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
	// HTTP3xxLocation url from HTTP response code 3xx
	HTTP3xxLocation
)

type urlContext int
