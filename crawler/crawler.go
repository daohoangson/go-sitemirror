package crawler

import (
	"errors"
	"net/http"
	neturl "net/url"
	"sync"
)

type crawler struct {
	client            *http.Client
	autoDownloadDepth int
	workerCount       int

	onURLShouldQueue *func(*neturl.URL) bool
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
	url   *neturl.URL
	depth int
}

// New returns a new instance of the crawler
func New(client *http.Client) Crawler {
	c := &crawler{}
	c.init(client)
	return c
}

func (c *crawler) init(client *http.Client) {
	if client == nil {
		client = http.DefaultClient
	}
	c.client = client

	c.autoDownloadDepth = 1
	c.workerCount = 4
}

func (c *crawler) SetAutoDownloadDepth(depth int) {
	c.autoDownloadDepth = depth
}

func (c *crawler) GetAutoDownloadDepth() int {
	return c.autoDownloadDepth
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

func (c *crawler) SetOnURLShouldQueue(f func(*neturl.URL) bool) {
	c.onURLShouldQueue = &f
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
							url:   foundURL,
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

func (c *crawler) Download(url *neturl.URL) {
	c.Start()
	c.queue <- queueItem{url: url}
	c.queuedCount++
}

func (c *crawler) DownloadURL(url string) error {
	parsedURL, err := neturl.Parse(url)
	if err != nil {
		return err
	}

	c.Download(parsedURL)
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
		return nil
	}
}
