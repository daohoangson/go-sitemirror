package crawler

import (
	"bytes"
	"net/http"
	"net/url"

	"github.com/Sirupsen/logrus"
	"time"
)

// Crawler represents an object that can process download requests
type Crawler interface {
	init(*http.Client, *logrus.Logger)

	GetClientTimeout() time.Duration
	SetAutoDownloadDepth(uint64)
	GetAutoDownloadDepth() uint64
	SetNoCrossHost(bool)
	GetNoCrossHost() bool
	AddRequestHeader(string, string)
	SetRequestHeader(string, string)
	GetRequestHeaderValues(string) []string
	SetWorkerCount(uint64) error
	GetWorkerCount() uint64

	SetURLRewriter(func(*url.URL))
	SetOnURLShouldQueue(func(*url.URL) bool)
	SetOnURLShouldDownload(func(*url.URL) bool)
	SetOnDownload(func(*url.URL))
	SetOnDownloaded(func(*Downloaded))

	GetEnqueuedCount() uint64
	GetDownloadedCount() uint64
	GetLinkFoundCount() uint64
	HasStarted() bool
	HasStopped() bool
	IsRunning() bool
	IsBusy() bool

	Start()
	Stop()
	Enqueue(QueueItem)
	Download(QueueItem) *Downloaded
	Downloaded() (*Downloaded, bool)
	DownloadedNotBlocking() *Downloaded
}

// QueueItem represents a download request in the queue
type QueueItem struct {
	URL           *url.URL
	Depth         uint64
	ForceDownload bool
}

// Input represents a download request ready to be processed
type Input struct {
	Client      *http.Client
	Header      http.Header
	NoCrossHost bool
	Rewriter    *func(*url.URL)
	URL         *url.URL
}

// Downloaded represents processed data after downloading
type Downloaded struct {
	Input *Input

	BaseURL         *url.URL
	Body            string
	Error           error
	LinksAssets     map[string]Link
	LinksDiscovered map[string]Link
	StatusCode      int

	buffer                  *bytes.Buffer
	header                  http.Header
	addedHeaderCrossHostRef bool
}

// Link represents an extracted link from download result
type Link struct {
	Context urlContext
	URL     *url.URL
}

const (
	// CSSUri url from url()
	CSSUri urlContext = 1 + iota
	// HTMLTagA url from <a href=""></a>
	HTMLTagA
	// HTMLTagForm url from <form action="" />
	HTMLTagForm
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
