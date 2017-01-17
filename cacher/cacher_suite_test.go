package cacher_test

import (
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestCacher(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Cacher Suite")
}

func getHeaderValue(written string, headerKey string) string {
	lines := strings.Split(written, "\n")

	for _, line := range lines {
		if len(line) == 0 {
			// reached content, return asap
			return ""
		}

		colon := strings.Index(line, ":")
		if colon < 0 {
			continue
		}

		lineHeaderKey := line[:colon]
		if lineHeaderKey == headerKey {
			return strings.TrimSpace(line[colon+1:])
		}
	}

	// header not found
	return ""
}

func getContent(written string) string {
	sep := "\n\n"
	index := strings.Index(written, sep)
	if index == -1 {
		// content not found
		return ""
	}

	return written[index+len(sep):]
}
