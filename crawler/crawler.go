package crawler

import (
	"errors"
	"net/http"
	neturl "net/url"
	"sync"
	"time"

	"github.com/Sirupsen/logrus"
)

type crawler struct {
	client *http.Client
	logger *logrus.Logger

	autoDownloadDepth int
	workerCount       int

	onURLShouldQueue *func(*neturl.URL) bool
	onDownload       *func(*neturl.URL)
	onDownloaded     *func(*Downloaded)

	output          chan *Downloaded
	queue           chan queueItem
	workerStartOnce sync.Once
	workersRunning  bool
	queuedCount     int
	downloadedCount int
	linkFoundCount  int
}

type queueItem struct {
	url      *neturl.URL
	depth    int
	workerID int
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

func (c *crawler) SetAutoDownloadDepth(depth int) {
	c.logger.WithFields(logrus.Fields{
		"old": c.autoDownloadDepth,
		"new": depth,
	}).Info("Updating crawler auto download depth")

	c.autoDownloadDepth = depth
}

func (c *crawler) GetAutoDownloadDepth() int {
	return c.autoDownloadDepth
}

func (c *crawler) SetWorkerCount(count int) error {
	if count < 1 {
		return errors.New("workerCount must be greater than 1")
	}

	if c.workersRunning {
		return errors.New("Cannot SetWorkerCount after Start")
	}

	c.logger.WithFields(logrus.Fields{
		"old": c.workerCount,
		"new": count,
	}).Info("Updating crawler worker count")

	c.workerCount = count

	return nil
}

func (c *crawler) GetWorkerCount() int {
	return c.workerCount
}

func (c *crawler) SetOnURLShouldQueue(f func(*neturl.URL) bool) {
	c.onURLShouldQueue = &f
}

func (c *crawler) SetOnDownload(f func(*neturl.URL)) {
	c.onDownload = &f
}

func (c *crawler) SetOnDownloaded(f func(*Downloaded)) {
	if c.onDownloaded == nil && c.workersRunning {
		go func() {
			for {
				downloaded := c.NextOrNil()
				if downloaded == nil {
					break
				}

				f(downloaded)
			}
		}()
	}

	c.onDownloaded = &f
}

func (c *crawler) IsWorkersRunning() bool {
	return c.workersRunning
}

func (c *crawler) GetQueuedCount() int {
	return c.queuedCount
}

func (c *crawler) GetDownloadedCount() int {
	return c.downloadedCount
}

func (c *crawler) GetLinkFoundCount() int {
	return c.linkFoundCount
}

func (c *crawler) Start() {
	c.workerStartOnce.Do(func() {
		c.queue = make(chan queueItem)
		c.output = make(chan *Downloaded)
		requeue := make(chan queueItem)

		go func() {
			for item := range requeue {
				c.queue <- item
				c.queuedCount++

				c.logger.WithFields(logrus.Fields{
					"url":    item.url,
					"depth":  item.depth,
					"worker": item.workerID,
					"total":  c.queuedCount,
				}).Debug("Auto-queued")
			}
		}()

		for i := 0; i < c.workerCount; i++ {
			go func(workerID int) {
				for queuedItem := range c.queue {
					if c.onDownload != nil {
						(*c.onDownload)(queuedItem.url)
					}

					downloaded := c.doDownload(workerID, queuedItem)

					c.doAutoQueue(workerID, queuedItem, downloaded, requeue)
				}
			}(i + 1)
		}

		c.workersRunning = true
	})
}

func (c *crawler) Queue(url *neturl.URL) {
	c.Start()
	c.queue <- queueItem{url: url}
	c.queuedCount++

	c.logger.WithFields(logrus.Fields{
		"url":   url,
		"total": c.queuedCount,
	}).Debug("Queued")
}

func (c *crawler) QueueURL(url string) error {
	parsedURL, err := neturl.Parse(url)
	if err != nil {
		c.logger.WithField("url", url).Error("Cannot download invalid url")

		return err
	}

	c.Queue(parsedURL)
	return nil
}

func (c *crawler) Next() *Downloaded {
	c.Start()
	result := <-c.output
	return result
}

func (c *crawler) NextOrNil() *Downloaded {
	c.Start()
	select {
	case result, _ := <-c.output:
		return result
	default:
		c.logger.Debug("No result in output channel")

		return nil
	}
}

func (c *crawler) doDownload(workerID int, item queueItem) *Downloaded {
	start := time.Now()
	c.logger.WithFields(logrus.Fields{
		"worker":   workerID,
		"url":      item.url,
		"queuedBy": item.workerID,
	}).Debug("Downloading")

	downloaded := Download(c.client, item.url)

	c.downloadedCount++
	c.logger.WithFields(logrus.Fields{
		"worker":     workerID,
		"url":        downloaded.URL,
		"statusCode": downloaded.StatusCode,
		"elapsed":    time.Since(start),
		"total":      c.downloadedCount,
	}).Info("Downloaded")

	if c.onDownloaded == nil {
		c.output <- downloaded
	} else {
		(*c.onDownloaded)(downloaded)
	}

	return downloaded
}

func (c *crawler) doAutoQueue(workerID int, item queueItem, downloaded *Downloaded, requeue chan<- queueItem) {
	linksLength := len(downloaded.Links)
	if linksLength == 0 {
		return
	}

	c.linkFoundCount += linksLength

	nextDepth := item.depth + 1
	if nextDepth > c.autoDownloadDepth {
		c.logger.WithFields(logrus.Fields{
			"worker": workerID,
			"source": downloaded.URL,
			"links":  linksLength,
			"depth":  nextDepth,
		}).Info("Skipped because they are too deep")
		return
	}

	foundUrls := downloaded.GetResolvedURLs()

	for _, foundURL := range foundUrls {
		if c.onURLShouldQueue != nil {
			shouldQueue := (*c.onURLShouldQueue)(foundURL)
			if !shouldQueue {
				c.logger.WithFields(logrus.Fields{
					"worker": workerID,
					"url":    foundURL,
					"depth":  nextDepth,
				}).Debug("Skipped as instructed by onURLShouldQueue")
				continue
			}
		}

		newItem := queueItem{
			url:      foundURL,
			depth:    nextDepth,
			workerID: workerID,
		}
		requeue <- newItem
	}
}
