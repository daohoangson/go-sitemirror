package crawler_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestCrawler(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Crawler Suite")
}
