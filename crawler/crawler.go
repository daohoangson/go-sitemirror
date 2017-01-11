package crawler

import (
	"errors"
	"net/http"
	"net/url"
	"sync"
)

type crawler struct {
	client            *http.Client
	autoDownloadDepth int
	workerCount       int

	onDownloaded     *downloadedFunc
	onURLShouldQueue *urlShouldQueueFunc

	output          downloadedChan
	queue           queueChan
	workerStartOnce sync.Once
	workersRunning  bool
	queuedCount     int
	downloadedCount int
	linkFoundCount  int
}

type queueItem struct {
	url   string
	depth int
}

type downloadedChan chan *Downloaded
type queueChan chan queueItem
type workerFunc func(*crawler)
type urlShouldQueueFunc func(*url.URL) bool
type downloadedFunc func(*Downloaded)

// Crawl creates a new instance and start crawling the specified urls
func Crawl(client *http.Client, urls ...string) Crawler {
	c := New(client)

	if len(urls) > 0 {
		c.SetWorkerCount(len(urls))
		for _, url := range urls {
			c.Download(url)
		}
	}

	return c
}

// New returns a new instance of the crawler
func New(client *http.Client) Crawler {
	c := &crawler{}
	c.init(client)
	return c
}

func (c *crawler) init(client *http.Client) {
	c.client = client

	c.autoDownloadDepth = 1
	c.workerCount = 4
}

func (c *crawler) SetWorkerCount(workerCount int) error {
	if workerCount < 1 {
		return errors.New("workerCount must be greater than 1")
	}

	if c.workersRunning {
		return errors.New("Cannot SetWorkerCount after Start")
	}

	c.workerCount = workerCount
	return nil
}

func (c *crawler) GetWorkerCount() int {
	return c.workerCount
}

func (c *crawler) SetOnDownloaded(f downloadedFunc) {
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

func (c *crawler) SetOnURLShouldQueue(f urlShouldQueueFunc) {
	c.onURLShouldQueue = &f
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
		c.queue = make(queueChan)
		c.output = make(downloadedChan)
		requeue := make(queueChan)

		go func() {
			for item := range requeue {
				c.queue <- item
				c.queuedCount++
			}
		}()

		for i := 0; i < c.workerCount; i++ {
			go func() {
				for queuedItem := range c.queue {
					downloaded := Download(c.client, queuedItem.url)
					c.downloadedCount++

					if c.onDownloaded == nil {
						c.output <- downloaded
					} else {
						(*c.onDownloaded)(downloaded)
					}

					c.linkFoundCount += len(downloaded.Links)

					linkDepth := queuedItem.depth + 1
					if linkDepth > c.autoDownloadDepth {
						continue
					}

					for i := 0; i < len(downloaded.Links); i++ {
						foundURL := downloaded.GetResolvedURL(i)

						if c.onURLShouldQueue != nil {
							shouldQueue := (*c.onURLShouldQueue)(foundURL)
							if !shouldQueue {
								continue
							}
						}

						newItem := queueItem{
							url:   foundURL.String(),
							depth: linkDepth,
						}
						requeue <- newItem
					}
				}
			}()
		}

		c.workersRunning = true
	})
}

func (c *crawler) Download(url string) {
	c.Start()
	c.queue <- queueItem{url: url}
	c.queuedCount++
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
		return nil
	}
}
