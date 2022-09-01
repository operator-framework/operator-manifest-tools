package imageresolver

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("getName func", func() {
	It("should parse image ref correctly", func() {
		Expect(getName("localhost:5005/controller:v0.0.1")).To(Equal("localhost:5005/controller"))
	})
})
