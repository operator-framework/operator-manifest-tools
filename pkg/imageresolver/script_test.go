package imageresolver

import (
	"io/ioutil"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("script image resolver", func() {
	var sut *Script
	var goodScript string
	var badScript string

	BeforeEach(func() {
		dir, err := ioutil.TempDir("", "script")
		Expect(err).To(Succeed())

		goodScript = filepath.Join(dir, "good.sh")
		badScript = filepath.Join(dir, "bad.sh")
		Expect(ioutil.WriteFile(goodScript, []byte(`#!/bin/bash
echo -n "foo"
exit 0
`), 0700)).To(Succeed())

		Expect(ioutil.WriteFile(badScript, []byte(`#!/bin/bash
exit 1
`), 0700)).To(Succeed())
	})

	It("should return results", func() {
		sut = &Script{path: goodScript}
		result, err := sut.ResolveImageReference("test")
		Expect(err).To(Succeed())
		Expect(result).Should(Equal("test@sha256:foo"))
	})

	It("should fail", func() {
		sut = &Script{path: badScript}
		_, err := sut.ResolveImageReference("test")
		Expect(err).To(HaveOccurred())
	})
})
