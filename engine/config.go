package engine

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/namsral/flag"
)

type Config struct {
	LoggerLevel configLoggerLevel

	HostRewrites        configStringMap
	HostsWhitelist      configStringSlice
	BumpTTL             time.Duration
	AutoEnqueueInterval time.Duration

	Cacher  configCacher
	Crawler configCrawler

	MirrorURLs  configURLSlice
	MirrorPorts configIntSlice
}

type configCacher struct {
	Path       string
	DefaultTTL time.Duration
}

type configCrawler struct {
	AutoDownloadDepth configUint64
	WorkerCount       configUint64
	RequestHeader     configHTTPHeader
}

type configHTTPHeader http.Header
type configLoggerLevel logrus.Level
type configIntSlice []int
type configStringMap map[string]string
type configStringSlice []string
type configUint64 uint64
type configURLSlice []*url.URL

const (
	ConfigEnvVarPrefix                    = "SITEMIRROR"
	ConfigDefaultLoggerLevel              = logrus.InfoLevel
	ConfigDefaultBumpTTL                  = time.Minute
	ConfigDefaultAutoEnqueueInterval      = time.Duration(0)
	ConfigDefaultCacherDefaultTTL         = 10 * time.Minute
	ConfigDefaultCrawlerAutoDownloadDepth = uint64(1)
	ConfigDefaultCrawlerWorkerCount       = uint64(4)
)

func ParseConfig(arg0 string, otherArgs []string) (*Config, error) {
	config := &Config{}

	fs := flag.NewFlagSetWithEnvPrefix(arg0, ConfigEnvVarPrefix, 0)

	config.LoggerLevel = configLoggerLevel(ConfigDefaultLoggerLevel)
	fs.Var(&config.LoggerLevel, "log", "Logging output level")

	fs.Var(&config.HostRewrites, "rewrite", "Link rewrites, must be 'source.domain.com=target.domain.com'")
	fs.Var(&config.HostsWhitelist, "whitelist", "Restricted list of crawlable hosts")
	fs.DurationVar(&config.BumpTTL, "cache-bump", ConfigDefaultBumpTTL, "Validity of cache bump, default=1m")
	fs.DurationVar(&config.AutoEnqueueInterval, "auto-refresh", ConfigDefaultAutoEnqueueInterval, "Interval for url auto refreshes, default=no refresh")

	fs.StringVar(&config.Cacher.Path, "cache-path", "", "HTTP Cache path, default = current directory")
	fs.DurationVar(&config.Cacher.DefaultTTL, "cache-ttl", ConfigDefaultCacherDefaultTTL, "Validity of cached data, default=10m")

	config.Crawler.AutoDownloadDepth = configUint64(ConfigDefaultCrawlerAutoDownloadDepth)
	fs.Var(&config.Crawler.AutoDownloadDepth, "auto-download-depth", "Maximum link depth for auto downloads, default=1")
	config.Crawler.WorkerCount = configUint64(ConfigDefaultCrawlerWorkerCount)
	fs.Var(&config.Crawler.WorkerCount, "workers", "Number of download workers, default=4")
	fs.Var(&config.Crawler.RequestHeader, "header", "Custom request header, must be 'key=value'")

	fs.Var(&config.MirrorURLs, "mirror", "URL to mirror, multiple urls are supported")
	fs.Var(&config.MirrorPorts, "port", "Port to mirror, each port number should immediately follow its URL. "+
		"For url that doesn't have any port, it will still be mirrored but without a web server.")

	err := fs.Parse(otherArgs)

	return config, err
}

func FromConfig(config *Config) Engine {
	logger := logrus.New()
	logger.Level = logrus.Level(config.LoggerLevel)

	e := New(http.DefaultClient, logger)

	{
		if config.HostRewrites != nil {
			hostRewrites := map[string]string(config.HostRewrites)
			for from, to := range hostRewrites {
				e.AddHostRewrite(from, to)
			}
		}

		if config.HostsWhitelist != nil {
			hostsWhitelist := []string(config.HostsWhitelist)
			for _, host := range hostsWhitelist {
				e.AddHostWhitelisted(host)
			}
		}

		e.SetBumpTTL(config.BumpTTL)
		e.SetAutoEnqueueInterval(config.AutoEnqueueInterval)
	}

	{
		cacher := e.GetCacher()
		if len(config.Cacher.Path) > 0 {
			cacher.SetPath(config.Cacher.Path)
		}
		cacher.SetDefaultTTL(config.Cacher.DefaultTTL)
	}

	{
		crawler := e.GetCrawler()
		crawler.SetAutoDownloadDepth(uint64(config.Crawler.AutoDownloadDepth))
		crawler.SetWorkerCount(uint64(config.Crawler.WorkerCount))

		if config.Crawler.RequestHeader != nil {
			requestHeader := http.Header(config.Crawler.RequestHeader)
			for headerKey, headerValues := range requestHeader {
				for _, headerValue := range headerValues {
					crawler.AddRequestHeader(headerKey, headerValue)
				}
			}
		}
	}

	{
		if config.MirrorURLs != nil {
			mirrorURLs := []*url.URL(config.MirrorURLs)
			mirrorPorts := []int(config.MirrorPorts)
			for i, url := range mirrorURLs {
				port := -1
				if i < len(mirrorPorts) {
					port = mirrorPorts[i]
				}

				e.Mirror(url, port)
			}
		}
	}

	return e
}

func (f *configHTTPHeader) String() string {
	return fmt.Sprint(*f)
}

func (f *configHTTPHeader) Set(value string) error {
	var (
		sep  = "="
		help = errors.New("must be 'key=value'")
	)

	parts := strings.Split(value, sep)
	if len(parts) < 2 {
		return help
	}
	key, value := parts[0], strings.Join(parts[1:], sep)

	if *f == nil {
		*f = make(configHTTPHeader)
	}

	http.Header(*f).Add(key, value)
	return nil
}

func (f *configLoggerLevel) String() string {
	return fmt.Sprint(*f)
}

func (f *configLoggerLevel) Set(value string) error {
	var (
		maxValue = len(logrus.AllLevels) - 1
		help     = fmt.Errorf("must be in range [0..%d] or "+
			"a level name ('debug', 'info', etc.)", maxValue)
	)

	parsedInt64, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		parsedLevel, err := logrus.ParseLevel(value)
		if err != nil {
			return help
		}

		*f = configLoggerLevel(parsedLevel)
		return nil
	}

	parsedInt := int(parsedInt64)
	if parsedInt < 0 || parsedInt > maxValue {
		return help
	}

	*f = configLoggerLevel(logrus.AllLevels[parsedInt])
	return nil
}

func (f *configIntSlice) String() string {
	return fmt.Sprint(*f)
}

func (f *configIntSlice) Set(value string) error {
	parsedInt64, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		return err
	}

	(*f) = append(*f, int(parsedInt64))
	return nil
}

func (f *configStringMap) String() string {
	return fmt.Sprint(*f)
}

func (f *configStringMap) Set(value string) error {
	var (
		help = errors.New("must be 'source.domain.com=target.domain.com'")
	)

	parts := strings.Split(value, "=")
	if len(parts) != 2 {
		return help
	}
	from, to := parts[0], parts[1]

	if *f == nil {
		*f = make(configStringMap)
	}

	(*f)[from] = to
	return nil
}

func (f *configStringSlice) String() string {
	return fmt.Sprint(*f)
}

func (f *configStringSlice) Set(value string) error {
	(*f) = append(*f, value)
	return nil
}

func (f *configUint64) String() string {
	return fmt.Sprint(*f)
}

func (f *configUint64) Set(value string) error {
	parsedUint64, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return err
	}

	*f = configUint64(parsedUint64)
	return nil
}

func (f *configURLSlice) String() string {
	return fmt.Sprint(*f)
}

func (f *configURLSlice) Set(value string) error {
	parsedURL, err := url.Parse(value)
	if err != nil {
		return err
	}

	(*f) = append(*f, parsedURL)
	return nil
}
