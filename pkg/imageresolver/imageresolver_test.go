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

var _ = Describe("GetResolver", func() {
	Describe("CraneAuth", func() {
		It("returns a Crane resolver with the default options", func() {
			args := make(map[string]string)
			args["usedefault"] = "true"

			resolver, err := GetResolver(ResolverCrane, args)
			Expect(err).To(BeNil())
			Expect(resolver).NotTo(BeNil())
			Expect(resolver.(CraneResolver).useDefault).To(BeTrue())
		})
	})
})
