package crawler

import (
	"errors"
	"net/http"
	neturl "net/url"
	"sync"
	"sync/atomic"
	"time"

	nbc "github.com/hectane/go-nonblockingchan"

	"github.com/Sirupsen/logrus"
)

type crawler struct {
	client *http.Client
	logger *logrus.Logger

	autoDownloadDepth uint64
	workerCount       uint64

	onURLShouldQueue    *func(*neturl.URL) bool
	onURLShouldDownload *func(*neturl.URL) bool
	onDownload          *func(*neturl.URL)
	onDownloaded        *func(*Downloaded)

	output           chan *Downloaded
	queue            *nbc.NonBlockingChan
	workerStartOnce  sync.Once
	workersStarted   uint64
	workersRunning   int64
	enqueuedCount    uint64
	queuingCount     int64
	downloadingCount int64
	downloadedCount  uint64
	linkFoundCount   uint64
}

type queueItem struct {
	url   *neturl.URL
	depth uint64
}

// New returns a new instance of the crawler
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

	c.autoDownloadDepth = 1
	c.workerCount = 4

	if logger == nil {
		logger = logrus.New()
	}
	c.logger = logger
}

func (c *crawler) SetAutoDownloadDepth(depth uint64) {
	c.logger.WithFields(logrus.Fields{
		"old": c.autoDownloadDepth,
		"new": depth,
	}).Info("Updating crawler auto download depth")

	c.autoDownloadDepth = depth
}

func (c *crawler) GetAutoDownloadDepth() uint64 {
	return c.autoDownloadDepth
}

func (c *crawler) SetWorkerCount(count uint64) error {
	if count < 1 {
		return errors.New("workerCount must be greater than 1")
	}

	if c.HasStarted() {
		return errors.New("Cannot SetWorkerCount after Start")
	}

	c.logger.WithFields(logrus.Fields{
		"old": c.workerCount,
		"new": count,
	}).Info("Updating crawler worker count")

	c.workerCount = count

	return nil
}

func (c *crawler) GetWorkerCount() uint64 {
	return c.workerCount
}

func (c *crawler) SetOnURLShouldQueue(f func(*neturl.URL) bool) {
	c.onURLShouldQueue = &f
}

func (c *crawler) SetOnURLShouldDownload(f func(*neturl.URL) bool) {
	c.onURLShouldDownload = &f
}

func (c *crawler) SetOnDownload(f func(*neturl.URL)) {
	c.onDownload = &f
}

func (c *crawler) SetOnDownloaded(f func(*Downloaded)) {
	if c.onDownloaded == nil && c.IsRunning() {
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

	c.onDownloaded = &f
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

		c.queue = nbc.New()
		c.output = make(chan *Downloaded)

		for i := uint64(0); i < workerCount; i++ {
			go func(workerID uint64) {
				atomic.AddUint64(&c.workersStarted, 1)
				atomic.AddInt64(&c.workersRunning, 1)

				for {
					if v, ok := <-c.queue.Recv; ok {
						if item, ok := v.(queueItem); ok {
							if c.onDownload != nil {
								(*c.onDownload)(item.url)
							}

							downloaded := c.doDownload(workerID, item)

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

	close(c.output)
	close(c.queue.Send)
	c.logger.Info("Stopped crawler")
}

func (c *crawler) Queue(url *neturl.URL) {
	c.Start()
	c.doEnqueue(queueItem{url: url})

	c.logger.WithField("url", url).Info("Queued")
}

func (c *crawler) QueueURL(url string) error {
	parsedURL, err := neturl.Parse(url)
	if err != nil {
		c.logger.WithField("url", url).Error("Cannot queue invalid url")

		return err
	}

	c.Queue(parsedURL)
	return nil
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

func (c *crawler) doEnqueue(item queueItem) {
	atomic.AddUint64(&c.enqueuedCount, 1)
	atomic.AddInt64(&c.queuingCount, 1)

	c.queue.Send <- item
}

func (c *crawler) doDownload(workerID uint64, item queueItem) *Downloaded {
	var (
		start         = time.Now()
		loggerContext = c.logger.WithFields(logrus.Fields{
			"worker": workerID,
			"url":    item.url,
		})
		shouldDownload = true
		downloaded     *Downloaded
	)

	atomic.AddInt64(&c.downloadingCount, 1)
	atomic.AddInt64(&c.queuingCount, -1)

	if c.onURLShouldDownload != nil {
		shouldDownload = (*c.onURLShouldDownload)(item.url)
		if !shouldDownload {
			loggerContext.Debug("Skipped as instructed by onURLShouldDownload")
		}
	}

	if shouldDownload {
		loggerContext.Debug("Downloading")
		downloaded = Download(c.client, item.url)
		atomic.AddUint64(&c.downloadedCount, 1)
	}

	atomic.AddInt64(&c.downloadingCount, -1)

	if downloaded != nil {
		loggerContext.WithFields(logrus.Fields{
			"statusCode": downloaded.StatusCode,
			"elapsed":    time.Since(start),
			"total":      c.downloadedCount,
		}).Info("Downloaded")

		if c.onDownloaded == nil {
			c.output <- downloaded
		} else {
			(*c.onDownloaded)(downloaded)
		}
	}

	return downloaded
}

func (c *crawler) doAutoQueue(workerID uint64, item queueItem, downloaded *Downloaded) {
	if downloaded == nil {
		return
	}

	// use the same depth for asset links as they are required for proper rendering
	c.doAutoQueueURLs(workerID, downloaded.GetAssetURLs(), downloaded.URL, item.depth)

	// increase depth for other discovered links
	// they will need to satisfy depth limit before crawling
	c.doAutoQueueURLs(workerID, downloaded.GetDiscoveredURLs(), downloaded.URL, item.depth+1)
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

	if nextDepth > c.autoDownloadDepth {
		loggerContext.WithField("links", count).Info("Skipped because it is too deep")
		return
	}

	for _, url := range urls {
		if c.onURLShouldQueue != nil {
			shouldQueue := (*c.onURLShouldQueue)(url)
			if !shouldQueue {
				loggerContext.WithField("url", url).Debug("Skipped as instructed by onURLShouldQueue")
				continue
			}
		}

		newItem := queueItem{
			url:   url,
			depth: nextDepth,
		}
		c.doEnqueue(newItem)

		loggerContext.WithField("url", url).Debug("Auto-enqueued")
	}
}
