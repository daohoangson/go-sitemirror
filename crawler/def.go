package crawler

import (
	"bytes"
	"net/http"
	"net/url"

	"github.com/Sirupsen/logrus"
)

// Crawler must be created with New
type Crawler interface {
	init(*http.Client, *logrus.Logger)

	SetAutoDownloadDepth(uint64)
	GetAutoDownloadDepth() uint64
	SetWorkerCount(uint64) error
	GetWorkerCount() uint64

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
	EnqueueURL(string) error
	Download(QueueItem) *Downloaded
	Downloaded() (*Downloaded, bool)
	DownloadedNotBlocking() *Downloaded
}

type QueueItem struct {
	URL           *url.URL
	Depth         uint64
	ForceDownload bool
}

// Downloaded struct contains parsed data after downloading an url.
type Downloaded struct {
	BaseURL         *url.URL
	BodyString      string
	BodyBytes       []byte
	ContentType     string
	Error           error
	HeaderLocation  *url.URL
	LinksAssets     map[string]Link
	LinksDiscovered map[string]Link
	StatusCode      int
	URL             *url.URL

	buffer *bytes.Buffer
}

// Link struct is an extracted link from download result.
type Link struct {
	Context urlContext
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
