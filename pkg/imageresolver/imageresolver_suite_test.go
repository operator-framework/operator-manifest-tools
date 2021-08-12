package imageresolver_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestImageresolver(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Imageresolver Suite")
}
