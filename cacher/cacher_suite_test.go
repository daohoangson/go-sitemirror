package cacher_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestCacher(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Cacher Suite")
}
