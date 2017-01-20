package engine_test

import (
	"fmt"
	"net/http"
	"os"
	"path"
	"time"

	httpmock "gopkg.in/jarcoal/httpmock.v1"

	"github.com/Sirupsen/logrus"
	. "github.com/daohoangson/go-sitemirror/engine"
	t "github.com/daohoangson/go-sitemirror/testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Config", func() {
	parseConfigWithDefaultArg0 := func(args ...string) *Config {
		config, _ := ParseConfig(os.Args[0], args)
		return config
	}

	Describe("ParseConfig", func() {
		It("should work without any args", func() {
			c := parseConfigWithDefaultArg0()

			Expect(c).ToNot(BeNil())
		})

		Describe("LoggerLevel", func() {
			It("should parse string", func() {
				c := parseConfigWithDefaultArg0("-log", "debug")

				Expect(c.LoggerLevel).To(BeNumerically("==", logrus.DebugLevel))
			})

			It("should parse int", func() {
				c := parseConfigWithDefaultArg0("-log", "1")

				Expect(c.LoggerLevel).To(BeNumerically("==", logrus.AllLevels[1]))
			})

			It("should handle int conversion error", func() {
				c := parseConfigWithDefaultArg0("-log", "x")

				Expect(c.LoggerLevel).To(BeNumerically("==", ConfigDefaultLoggerLevel))
			})

			It("should handle out of range (negative)", func() {
				c := parseConfigWithDefaultArg0("-log", "-1")

				Expect(c.LoggerLevel).To(BeNumerically("==", ConfigDefaultLoggerLevel))
			})

			It("should handle out of range (too large)", func() {
				c := parseConfigWithDefaultArg0("-log", fmt.Sprintf("%d", len(logrus.AllLevels)))

				Expect(c.LoggerLevel).To(BeNumerically("==", ConfigDefaultLoggerLevel))
			})
		})

		Describe("HostRewrites", func() {
			It("should parse", func() {
				c := parseConfigWithDefaultArg0("-rewrite", "domain2.com=domain.com")

				Expect(len(c.HostRewrites)).To(Equal(1))
				Expect(c.HostRewrites["domain2.com"]).To(Equal("domain.com"))
			})

			It("should parse multiple", func() {
				c := parseConfigWithDefaultArg0(
					"-rewrite", "domain2.com=domain.com",
					"-rewrite", "domain3.com=domain.com",
				)

				Expect(len(c.HostRewrites)).To(Equal(2))
				Expect(c.HostRewrites["domain2.com"]).To(Equal("domain.com"))
				Expect(c.HostRewrites["domain3.com"]).To(Equal("domain.com"))
			})

			It("should handle value in wrong format", func() {
				c := parseConfigWithDefaultArg0("-rewrite", "nop")

				Expect(c.HostRewrites).To(BeNil())
			})
		})

		Describe("HostsWhitelist", func() {
			It("should parse", func() {
				c := parseConfigWithDefaultArg0("-whitelist", "domain.com")

				Expect(len(c.HostsWhitelist)).To(Equal(1))
				Expect(c.HostsWhitelist[0]).To(Equal("domain.com"))
			})

			It("should parse multiple", func() {
				c := parseConfigWithDefaultArg0(
					"-whitelist", "domain.com",
					"-whitelist", "domain2.com",
				)

				Expect(len(c.HostsWhitelist)).To(Equal(2))
				Expect(c.HostsWhitelist[0]).To(Equal("domain.com"))
				Expect(c.HostsWhitelist[1]).To(Equal("domain2.com"))
			})
		})

		It("should parse BumpTTL", func() {
			c := parseConfigWithDefaultArg0("-cache-bump", "10ms")

			Expect(c.BumpTTL).To(Equal(10 * time.Millisecond))
		})

		It("should parse AutoEnqueueInterval", func() {
			c := parseConfigWithDefaultArg0("-auto-refresh", "1m")

			Expect(c.AutoEnqueueInterval).To(Equal(time.Minute))
		})

		Describe("Cacher", func() {
			It("should parse Path", func() {
				path := "cacher/path"
				c := parseConfigWithDefaultArg0("-cache-path", path)

				Expect(c.Cacher.Path).To(Equal(path))
			})

			It("should parse DefaultTTL", func() {
				c := parseConfigWithDefaultArg0("-cache-ttl", "10m")

				Expect(c.Cacher.DefaultTTL).To(Equal(10 * time.Minute))
			})
		})

		Describe("Crawler", func() {
			Describe("AutoDownloadDepth", func() {
				It("should parse", func() {
					c := parseConfigWithDefaultArg0("-auto-download-depth", "1")

					Expect(c.Crawler.AutoDownloadDepth).To(BeNumerically("==", 1))
				})

				It("should handle uint conversion error", func() {
					c := parseConfigWithDefaultArg0("-auto-download-depth", "x")

					Expect(c.Crawler.AutoDownloadDepth).To(BeNumerically("==", ConfigDefaultCrawlerAutoDownloadDepth))
				})
			})

			It("should parse NoCrossHost", func() {
				c := parseConfigWithDefaultArg0("-no-cross-host")

				Expect(c.Crawler.NoCrossHost).To(BeTrue())
			})

			Describe("RequestHeader", func() {
				It("should parse", func() {
					c := parseConfigWithDefaultArg0("-header", "key=value")

					Expect(len(c.Crawler.RequestHeader)).To(Equal(1))
					Expect(http.Header(c.Crawler.RequestHeader).Get("key")).To(Equal("value"))
				})

				It("should parse multiple", func() {
					c := parseConfigWithDefaultArg0(
						"-header", "key=value",
						"-header", "key2=value2",
					)

					Expect(len(c.Crawler.RequestHeader)).To(Equal(2))
					Expect(http.Header(c.Crawler.RequestHeader).Get("key")).To(Equal("value"))
					Expect(http.Header(c.Crawler.RequestHeader).Get("key2")).To(Equal("value2"))
				})

				It("should handle value in wrong format", func() {
					c := parseConfigWithDefaultArg0("-header", "nop")

					Expect(c.Crawler.RequestHeader).To(BeNil())
				})
			})

			Describe("WorkerCount", func() {
				It("should parse", func() {
					c := parseConfigWithDefaultArg0("-workers", "1")

					Expect(c.Crawler.WorkerCount).To(BeNumerically("==", 1))
				})

				It("should handle uint conversion error", func() {
					c := parseConfigWithDefaultArg0("-workers", "x")

					Expect(c.Crawler.WorkerCount).To(BeNumerically("==", ConfigDefaultCrawlerWorkerCount))
				})
			})
		})

		It("should parse Port", func() {
			c := parseConfigWithDefaultArg0("-port", "80")

			Expect(c.Port).To(Equal(int64(80)))
		})

		Describe("MirrorURLs", func() {
			It("should parse", func() {
				url := "http://domain.com"
				c := parseConfigWithDefaultArg0("-mirror", url)

				Expect(len(c.MirrorURLs)).To(Equal(1))
				Expect(c.MirrorURLs[0].String()).To(Equal(url))
			})

			It("should parse multiple", func() {
				url1 := "http://domain.com/one"
				url2 := "http://domain.com/two"
				c := parseConfigWithDefaultArg0("-mirror", url1, "-mirror", url2)

				Expect(len(c.MirrorURLs)).To(Equal(2))
				Expect(c.MirrorURLs[0].String()).To(Equal(url1))
				Expect(c.MirrorURLs[1].String()).To(Equal(url2))
			})

			It("should handle url parsing error", func() {
				c := parseConfigWithDefaultArg0("-mirror", t.InvalidURL)

				Expect(c.MirrorURLs).To(BeNil())
			})
		})

		Describe("MirrorPorts", func() {
			It("should parse", func() {
				c := parseConfigWithDefaultArg0("-mirror-port", "80")

				Expect(len(c.MirrorPorts)).To(Equal(1))
				Expect(c.MirrorPorts[0]).To(Equal(80))
			})

			It("should parse multiple", func() {
				c := parseConfigWithDefaultArg0("-mirror-port", "80", "-mirror-port", "81")

				Expect(len(c.MirrorPorts)).To(Equal(2))
				Expect(c.MirrorPorts[0]).To(Equal(80))
				Expect(c.MirrorPorts[1]).To(Equal(81))
			})

			It("should handle int parsing error", func() {
				c := parseConfigWithDefaultArg0("-mirror-port", "x")

				Expect(c.MirrorPorts).To(BeNil())
			})
		})
	})

	Describe("FromConfig", func() {
		It("should return", func() {
			config := parseConfigWithDefaultArg0()
			e := FromConfig(config)

			Expect(e).ToNot(BeNil())
		})

		It("should add host rewrite", func() {
			hostRewrites := make(map[string]string)
			hostRewrites["domain1.com"] = "domain.com"
			config := parseConfigWithDefaultArg0("-rewrite", "domain1.com=domain.com")
			e := FromConfig(config)

			Expect(e.GetHostRewrites()).To(Equal(hostRewrites))
		})

		It("should add host whitelisted", func() {
			hostsWhitelist := []string{"domain.com"}
			config := parseConfigWithDefaultArg0("-whitelist", hostsWhitelist[0])
			e := FromConfig(config)

			Expect(e.GetHostsWhitelist()).To(Equal(hostsWhitelist))
		})

		It("should set bump ttl", func() {
			ttl := time.Hour
			config := parseConfigWithDefaultArg0("-cache-bump", fmt.Sprintf("%s", ttl))
			e := FromConfig(config)

			Expect(e.GetBumpTTL()).To(Equal(ttl))
		})

		It("should set auto enqueue interval", func() {
			interval := time.Hour
			config := parseConfigWithDefaultArg0("-auto-refresh", fmt.Sprintf("%s", interval))
			e := FromConfig(config)

			Expect(e.GetAutoEnqueueInterval()).To(Equal(interval))
		})

		Describe("Cacher", func() {
			It("should set path", func() {
				path := "cacher/path"
				config := parseConfigWithDefaultArg0("-cache-path", path)
				e := FromConfig(config)

				Expect(e.GetCacher().GetPath()).To(Equal(path))
			})

			It("should set default ttl", func() {
				ttl := time.Hour
				config := parseConfigWithDefaultArg0("-cache-ttl", fmt.Sprintf("%s", ttl))
				e := FromConfig(config)

				Expect(e.GetCacher().GetDefaultTTL()).To(Equal(ttl))
			})
		})

		Describe("Crawler", func() {
			uint64Ten := uint64(10)

			It("should set auto download depth", func() {
				depth := uint64Ten
				config := parseConfigWithDefaultArg0("-auto-download-depth", fmt.Sprintf("%d", depth))
				e := FromConfig(config)

				Expect(e.GetCrawler().GetAutoDownloadDepth()).To(Equal(depth))
			})

			It("should set no cross host", func() {
				config := parseConfigWithDefaultArg0("-no-cross-host")
				e := FromConfig(config)

				Expect(e.GetCrawler().GetNoCrossHost()).To(BeTrue())
			})

			It("should add request header", func() {
				config := parseConfigWithDefaultArg0("-header", "key=value")
				e := FromConfig(config)

				Expect(e.GetCrawler().GetRequestHeaderValues("key")).To(Equal([]string{"value"}))
			})

			It("should set worker count", func() {
				workers := uint64Ten
				config := parseConfigWithDefaultArg0("-workers", fmt.Sprintf("%d", workers))
				e := FromConfig(config)

				Expect(e.GetCrawler().GetWorkerCount()).To(Equal(workers))
			})
		})

		Describe("Mirror", func() {
			const sleepTime = 20 * time.Millisecond
			const uint64One = uint64(1)
			const uint64Two = uint64(2)

			tmpDir := os.TempDir()
			rootPath := path.Join(tmpDir, "_TestFromConfig_")

			BeforeEach(func() {
				httpmock.Activate()
				os.Mkdir(rootPath, os.ModePerm)
			})

			AfterEach(func() {
				os.RemoveAll(rootPath)
				httpmock.DeactivateAndReset()
			})

			It("should mirror cross-host", func() {
				config := parseConfigWithDefaultArg0(
					"-cache-path", rootPath,
					"-port", "0",
				)

				e := FromConfig(config)
				defer e.Stop()

				port, _ := e.GetServer().GetListeningPort("")
				Expect(port).To(BeNumerically(">", 0))
			})

			It("should mirror url", func() {
				url := "http://domain.com/engine/FromConfig/mirror/url"
				config := parseConfigWithDefaultArg0(
					"-cache-path", rootPath,
					"-mirror", url,
				)
				httpmock.RegisterResponder("GET", url, httpmock.NewStringResponder(200, ""))

				e := FromConfig(config)
				defer e.Stop()

				time.Sleep(sleepTime)
				Expect(e.GetCrawler().GetDownloadedCount()).To(Equal(uint64One))

				port, _ := e.GetServer().GetListeningPort("domain.com")
				Expect(port).To(Equal(0))
			})

			It("should mirror with port", func() {
				url := "http://domain.com/engine/FromConfig/mirror/with/port"
				config := parseConfigWithDefaultArg0(
					"-cache-path", rootPath,
					"-mirror", url,
					"-mirror-port", "0",
				)
				httpmock.RegisterResponder("GET", url, httpmock.NewStringResponder(200, ""))

				e := FromConfig(config)
				defer e.Stop()

				time.Sleep(sleepTime)
				Expect(e.GetCrawler().GetDownloadedCount()).To(Equal(uint64One))

				port, _ := e.GetServer().GetListeningPort("domain.com")
				Expect(port).To(BeNumerically(">", 0))
			})

			It("should mirror multiple", func() {
				url1 := "http://domain1.com/engine/FromConfig/mirror/multiple"
				url2 := "http://domain2.com/engine/FromConfig/mirror/multiple"
				config := parseConfigWithDefaultArg0(
					"-cache-path", rootPath,
					"-mirror", url1, "-mirror-port", "0",
					"-mirror", url2,
				)
				httpmock.RegisterResponder("GET", url1, httpmock.NewStringResponder(200, ""))
				httpmock.RegisterResponder("GET", url2, httpmock.NewStringResponder(200, ""))

				e := FromConfig(config)
				defer e.Stop()

				time.Sleep(sleepTime)
				Expect(e.GetCrawler().GetDownloadedCount()).To(Equal(uint64Two))

				port1, _ := e.GetServer().GetListeningPort("domain1.com")
				Expect(port1).To(BeNumerically(">", 0))

				port2, _ := e.GetServer().GetListeningPort("domain2.com")
				Expect(port2).To(Equal(0))
			})
		})
	})
})
