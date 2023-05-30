package crawler

import (
	"errors"
	"fmt"
	"net/http"
	neturl "net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/hectane/go-nonblockingchan"
	"github.com/tevino/abool"
)

var version = "unknown"

type crawler struct {
	client *http.Client
	logger *logrus.Logger
	mutex  sync.Mutex

	autoDownloadDepth uint64
	noCrossHost       *abool.AtomicBool
	requestHeader     http.Header
	workerCount       uint64

	urlRewriter         *func(*neturl.URL)
	onURLShouldQueue    *func(*neturl.URL) bool
	onURLShouldDownload *func(*neturl.URL) bool
	onDownload          *func(*neturl.URL)
	onDownloaded        *func(*Downloaded)

	output           chan *Downloaded
	queue            *nbc.NonBlockingChan
	queueOpen        bool
	workerStartOnce  sync.Once
	workersStarted   uint64
	workersRunning   int64
	enqueuedCount    uint64
	queuingCount     int64
	downloadingCount int64
	downloadedCount  uint64
	linkFoundCount   uint64
}

// New returns a new crawler instance
func New(client *http.Client, logger *logrus.Logger) Crawler {
	c := &crawler{}
	c.init(client, logger)
	return c
}

func (c *crawler) init(client *http.Client, logger *logrus.Logger) {
	if client == nil {
		client = http.DefaultClient
	}
	c.client = client

	if logger == nil {
		logger = logrus.New()
	}
	c.logger = logger

	c.autoDownloadDepth = 1
	c.noCrossHost = abool.New()
	c.requestHeader = make(http.Header)
	c.workerCount = 4

	userAgent := fmt.Sprintf("go-sitemirror/%s (Googlebot wannabe)", version)
	c.requestHeader.Add("User-Agent", userAgent)
	logger.WithFields(logrus.Fields{
		"clientTimeout": client.Timeout,
		"userAgent":     userAgent,
	}).Debug("Initialized crawler")
}

func (c *crawler) GetClientTimeout() time.Duration {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	return c.client.Timeout
}

func (c *crawler) SetAutoDownloadDepth(depth uint64) {
	old := atomic.LoadUint64(&c.autoDownloadDepth)
	atomic.StoreUint64(&c.autoDownloadDepth, depth)

	c.logger.WithFields(logrus.Fields{
		"old": old,
		"new": depth,
	}).Info("Updated crawler auto download depth")
}

func (c *crawler) GetAutoDownloadDepth() uint64 {
	return atomic.LoadUint64(&c.autoDownloadDepth)
}

func (c *crawler) SetNoCrossHost(value bool) {
	old := c.noCrossHost.IsSet()
	c.noCrossHost.SetTo(value)

	c.logger.WithFields(logrus.Fields{
		"old": old,
		"new": value,
	}).Info("Updated crawler no cross host")
}

func (c *crawler) GetNoCrossHost() bool {
	return c.noCrossHost.IsSet()
}

func (c *crawler) AddRequestHeader(key string, value string) {
	c.mutex.Lock()
	c.requestHeader.Add(key, value)
	c.mutex.Unlock()

	c.logger.WithFields(logrus.Fields{
		"key":    key,
		"value":  value,
		"header": c.requestHeader,
	}).Info("Added request header")
}

func (c *crawler) SetRequestHeader(key string, value string) {
	c.mutex.Lock()
	c.requestHeader.Set(key, value)
	c.mutex.Unlock()

	c.logger.WithFields(logrus.Fields{
		"key":    key,
		"value":  value,
		"header": c.requestHeader,
	}).Info("Set request header")
}

func (c *crawler) GetRequestHeaderValues(key string) []string {
	chk := http.CanonicalHeaderKey(key)

	c.mutex.Lock()
	defer c.mutex.Unlock()
	if values, ok := c.requestHeader[chk]; ok {
		return values
	}

	return nil
}

func (c *crawler) SetWorkerCount(count uint64) error {
	if count < 1 {
		return errors.New("workerCount must be greater than 1")
	}

	if c.HasStarted() {
		return errors.New("cannot SetWorkerCount after Start")
	}

	old := atomic.LoadUint64(&c.workerCount)
	atomic.StoreUint64(&c.workerCount, count)

	c.logger.WithFields(logrus.Fields{
		"old": old,
		"new": count,
	}).Info("Updated crawler worker count")
	return nil
}

func (c *crawler) GetWorkerCount() uint64 {
	return atomic.LoadUint64(&c.workerCount)
}

func (c *crawler) SetURLRewriter(f func(*neturl.URL)) {
	c.mutex.Lock()
	c.urlRewriter = &f
	c.mutex.Unlock()
}

func (c *crawler) SetOnURLShouldQueue(f func(*neturl.URL) bool) {
	c.mutex.Lock()
	c.onURLShouldQueue = &f
	c.mutex.Unlock()
}

func (c *crawler) SetOnURLShouldDownload(f func(*neturl.URL) bool) {
	c.mutex.Lock()
	c.onURLShouldDownload = &f
	c.mutex.Unlock()
}

func (c *crawler) SetOnDownload(f func(*neturl.URL)) {
	c.mutex.Lock()
	c.onDownload = &f
	c.mutex.Unlock()
}

func (c *crawler) SetOnDownloaded(f func(*Downloaded)) {
	c.mutex.Lock()
	old := c.onDownloaded
	c.onDownloaded = &f
	c.mutex.Unlock()

	if old == nil && c.IsRunning() {
		go func() {
			for {
				downloaded := c.DownloadedNotBlocking()
				if downloaded == nil {
					break
				}

				f(downloaded)
			}
		}()
	}
}

func (c *crawler) GetEnqueuedCount() uint64 {
	return atomic.LoadUint64(&c.enqueuedCount)
}

func (c *crawler) GetDownloadedCount() uint64 {
	return atomic.LoadUint64(&c.downloadedCount)
}

func (c *crawler) GetLinkFoundCount() uint64 {
	return atomic.LoadUint64(&c.linkFoundCount)
}

func (c *crawler) HasStarted() bool {
	return atomic.LoadUint64(&c.workersStarted) > 0
}

func (c *crawler) HasStopped() bool {
	if !c.HasStarted() {
		return false
	}

	return !c.IsRunning()
}

func (c *crawler) IsRunning() bool {
	return atomic.LoadInt64(&c.workersRunning) > 0
}

func (c *crawler) IsBusy() bool {
	queuingCount := atomic.LoadInt64(&c.queuingCount)
	if queuingCount > 0 {
		c.logger.WithField("queuing", queuingCount).Debug("IsBusy")
		return true
	}

	downloadingCount := atomic.LoadInt64(&c.downloadingCount)
	if downloadingCount > 0 {
		c.logger.WithField("downloading", downloadingCount).Debug("IsBusy")
		return true
	}

	c.logger.Debug("IsBusyNot")
	return false
}

func (c *crawler) Start() {
	c.workerStartOnce.Do(func() {
		workerCount := atomic.LoadUint64(&c.workerCount)

		loggerContext := c.logger.WithFields(logrus.Fields{
			"workers": workerCount,
		})
		loggerContext.Debug("Starting crawler")

		c.mutex.Lock()
		c.queue = nbc.New()
		c.output = make(chan *Downloaded)
		c.queueOpen = true
		c.mutex.Unlock()

		for i := uint64(0); i < workerCount; i++ {
			go func(workerID uint64) {
				atomic.AddUint64(&c.workersStarted, 1)
				atomic.AddInt64(&c.workersRunning, 1)

				for {
					if v, ok := <-c.queue.Recv; ok {
						if item, ok := v.(QueueItem); ok {
							downloaded := c.doDownload(item)

							c.doAutoQueue(workerID, item, downloaded)
						}
					} else {
						break
					}
				}

				atomic.AddInt64(&c.workersRunning, -1)
			}(i + 1)
		}

		loggerContext.Info("Started crawler")
	})
}

func (c *crawler) Stop() {
	if !c.HasStarted() {
		c.logger.Debug("Crawler hasn't started")
		return
	}

	if c.HasStopped() {
		c.logger.Debug("Crawler has already stopped")
		return
	}

	c.mutex.Lock()
	c.queueOpen = false
	close(c.output)
	close(c.queue.Send)
	c.mutex.Unlock()

	c.logger.Info("Stopped crawler")
}

func (c *crawler) Enqueue(item QueueItem) {
	c.Start()
	c.doEnqueue(item)
}

func (c *crawler) Download(item QueueItem) *Downloaded {
	return c.doDownload(item)
}

func (c *crawler) Downloaded() (*Downloaded, bool) {
	c.Start()
	result, ok := <-c.output
	return result, ok
}

func (c *crawler) DownloadedNotBlocking() *Downloaded {
	c.Start()
	select {
	case result, _ := <-c.output:
		return result
	default:
		c.logger.Debug("No result in output channel")

		return nil
	}
}

func (c *crawler) doEnqueue(item QueueItem) {
	atomic.AddUint64(&c.enqueuedCount, 1)
	atomic.AddInt64(&c.queuingCount, 1)

	c.mutex.Lock()
	if c.queueOpen {
		c.queue.Send <- item
	}
	c.mutex.Unlock()

	c.logger.WithField("item", item).Debug("Enqueued")
}

func (c *crawler) doDownload(item QueueItem) *Downloaded {
	var (
		start          = time.Now()
		loggerContext  = c.logger.WithField("item", item)
		shouldDownload = true
		downloaded     *Downloaded
	)

	c.mutex.Lock()
	client := c.client
	requestHeader := c.requestHeader
	urlRewriter := c.urlRewriter
	onDownload := c.onDownload
	onURLShouldDownload := c.onURLShouldDownload
	onDownloaded := c.onDownloaded
	c.mutex.Unlock()

	if onDownload != nil {
		(*onDownload)(item.URL)
	}

	atomic.AddInt64(&c.downloadingCount, 1)
	atomic.AddInt64(&c.queuingCount, -1)

	if item.ForceDownload {
		// do not trigger onURLShouldDownload
	} else if onURLShouldDownload != nil {
		shouldDownload = (*onURLShouldDownload)(item.URL)
		if !shouldDownload {
			loggerContext.Debug("Skipped as instructed by onURLShouldDownload")
		}
	}

	if shouldDownload {
		loggerContext.Debug("Downloading")
		downloaded = Download(&Input{
			Client:      client,
			Header:      requestHeader,
			NoCrossHost: c.noCrossHost.IsSet(),
			Rewriter:    urlRewriter,
			URL:         item.URL,
		})
		atomic.AddUint64(&c.downloadedCount, 1)
	}

	atomic.AddInt64(&c.downloadingCount, -1)

	if downloaded != nil {
		if downloaded.Error != nil {
			loggerContext.WithFields(logrus.Fields{
				"error":   downloaded.Error,
				"elapsed": time.Since(start),
			}).Error("Error downloading")
		} else {
			loggerContext.WithFields(logrus.Fields{
				"statusCode": downloaded.StatusCode,
				"elapsed":    time.Since(start),
				"total":      atomic.LoadUint64(&c.downloadedCount),
			}).Info("Downloaded")
		}

		if onDownloaded != nil {
			(*onDownloaded)(downloaded)
		} else if c.IsRunning() {
			c.output <- downloaded
		}
	}

	return downloaded
}

func (c *crawler) doAutoQueue(workerID uint64, item QueueItem, downloaded *Downloaded) {
	if downloaded == nil {
		return
	}

	// use the same depth for asset links as they are required for proper rendering
	c.doAutoQueueURLs(workerID, downloaded.GetAssetURLs(), downloaded.Input.URL, item.Depth)

	// increase depth for other discovered links
	// they will need to satisfy depth limit before crawling
	c.doAutoQueueURLs(workerID, downloaded.GetDiscoveredURLs(), downloaded.Input.URL, item.Depth+1)
}

func (c *crawler) doAutoQueueURLs(workerID uint64, urls []*neturl.URL, source *neturl.URL, nextDepth uint64) {
	var (
		count         = len(urls)
		loggerContext = c.logger.WithFields(logrus.Fields{
			"worker": workerID,
			"source": source,
			"depth":  nextDepth,
		})
	)

	if count == 0 {
		return
	}

	atomic.AddUint64(&c.linkFoundCount, uint64(count))
	c.mutex.Lock()
	onURLShouldQueue := c.onURLShouldQueue
	c.mutex.Unlock()

	if nextDepth > c.autoDownloadDepth {
		loggerContext.WithField("links", count).Debug("Skipped because it is too deep")
		return
	}

	for _, url := range urls {
		if onURLShouldQueue != nil {
			shouldQueue := (*onURLShouldQueue)(url)
			if !shouldQueue {
				loggerContext.WithField("url", url).Debug("Skipped as instructed by onURLShouldQueue")
				continue
			}
		}

		c.doEnqueue(QueueItem{
			URL:   url,
			Depth: nextDepth,
		})

		loggerContext.WithField("url", url).Debug("Auto-enqueued")
	}
}
