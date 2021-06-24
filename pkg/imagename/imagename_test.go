package imagename

import (
	"fmt"

	. "github.com/onsi/ginkgo/extensions/table"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ImageNameParse", func() {
	DescribeTable("parses",
		func(text string, namedImage *ImageName) {
			Expect(*Parse(text)).To(Equal(*namedImage))
			Expect(namedImage.String()).To(Equal(text))
		},
		Entry("1", "repository.com/image-name:latest",
			&ImageName{Registry: "repository.com", Repo: "image-name", Tag: "latest"}),
		Entry("repository.com/prefix/image-name:1", "repository.com/prefix/image-name:1",
			&ImageName{Registry: "repository.com",
				Namespace: "prefix",
				Repo:      "image-name", Tag: "1"}),
		Entry("repository.com/prefix/image-name@sha256:12345", "repository.com/prefix/image-name@sha256:12345",
			&ImageName{Registry: "repository.com",
				Namespace: "prefix",
				Repo:      "image-name",
				Tag:       "sha256:12345"}),
		Entry("repository.com/prefix/image-name:latest", "repository.com/prefix/image-name:latest",
			&ImageName{Registry: "repository.com",
				Namespace: "prefix",
				Repo:      "image-name",
				Tag:       "latest",
			}),
		Entry("image-name:latest", "image-name:latest",
			&ImageName{Repo: "image-name", Tag: "latest"}),
		Entry("registry:5000/image-name@sha256:12345", "registry:5000/image-name@sha256:12345",
			&ImageName{Registry: "registry:5000", Repo: "image-name", Tag: "sha256:12345"}),
		Entry("registry:5000/image-name:latest", "registry:5000/image-name:latest",
			&ImageName{Registry: "registry:5000", Repo: "image-name", Tag: "latest"}),
		Entry("fedora:20", "fedora:20",
			&ImageName{Repo: "fedora", Tag: "20"}),
		Entry("fedora@sha256:12345", "fedora@sha256:12345",
			&ImageName{Repo: "fedora", Tag: "sha256:12345"}),
		Entry("prefix/image-name:1", "prefix/image-name:1",
			&ImageName{Namespace: "prefix", Repo: "image-name", Tag: "1"}),
		Entry("prefix/image-name@sha256:12345", "prefix/image-name@sha256:12345",
			&ImageName{Namespace: "prefix", Repo: "image-name",
				Tag: "sha256:12345"}),
		Entry("library/fedora:20", "library/fedora:20",
			&ImageName{Namespace: "library", Repo: "fedora", Tag: "20"}),
		Entry("library/fedora@sha256:12345", "library/fedora@sha256:12345",
			&ImageName{Namespace: "library", Repo: "fedora",
				Tag: "sha256:12345"}),
		Entry("registry.io/r-an3:1", "registry.io/r-an3:1",
			&ImageName{Registry: "registry.io", Repo: "r-an3", Tag: "1"}),
		Entry("registry.io/r-an3:1", "registry.io/r-an3:1",
			&ImageName{Registry: "registry.io", Repo: "r-an3", Tag: "1"},
		),
	)

	DescribeTable("encloses",
		func(repo, organization, enclosedRepo, registry, tag string) {
			reference := repo

			if tag != "" {
				reference = fmt.Sprintf("%s:%s", repo, tag)
			}

			if reference != "" {
				reference = fmt.Sprintf("%s/%s", registry, reference)
			}

			imageName := Parse(reference)

			Expect(imageName.GetRepo(DefaultGetStringOptions)).To(Equal(repo))
			Expect(imageName.Registry).To(Equal(registry))
			Expect(imageName.Tag).To(Or(Equal(tag), Equal("latest")))

			imageName.Enclose(organization)
			Expect(imageName.GetRepo(DefaultGetStringOptions)).To(Equal(enclosedRepo))
			Expect(imageName.Registry).To(Equal(registry))
			Expect(imageName.Tag).To(Or(Equal(tag), Equal("latest")))
		},
		Entry("1", "fedora", "spam", "spam/fedora", "example.registry.com", "bacon"),
		Entry("1", "fedora", "spam", "spam/fedora", "example.registry.com:8888", "bacon"),
		Entry("1", "fedora", "spam", "spam/fedora", "", "bacon"),
		Entry("1", "fedora", "spam", "spam/fedora", "example.registry.com", ""),
		Entry("1", "fedora", "spam", "spam/fedora", "example.registry.com:8888", ""),
		Entry("1", "fedora", "spam", "spam/fedora", "", ""),
		Entry("2", "spam/fedora", "spam", "spam/fedora", "example.registry.com", "bacon"),
		Entry("2", "spam/fedora", "spam", "spam/fedora", "example.registry.com:8888", "bacon"),
		Entry("2", "spam/fedora", "spam", "spam/fedora", "", "bacon"),
		Entry("2", "spam/fedora", "spam", "spam/fedora", "example.registry.com", ""),
		Entry("2", "spam/fedora", "spam", "spam/fedora", "example.registry.com:8888", ""),
		Entry("2", "spam/fedora", "spam", "spam/fedora", "", ""),
		Entry("3", "spam/fedora", "maps", "maps/spam-fedora", "example.registry.com", "bacon"),
		Entry("3", "spam/fedora", "maps", "maps/spam-fedora", "example.registry.com:8888", "bacon"),
		Entry("3", "spam/fedora", "maps", "maps/spam-fedora", "", "bacon"),
		Entry("3", "spam/fedora", "maps", "maps/spam-fedora", "example.registry.com", ""),
		Entry("3", "spam/fedora", "maps", "maps/spam-fedora", "example.registry.com:8888", ""),
		Entry("3", "spam/fedora", "maps", "maps/spam-fedora", "", ""),
	)

	It("should compare imagenames", func() {
		i1 := ImageName{Registry: "foo.com", Namespace: "spam", Repo: "bar", Tag: "1"}
		i2 := ImageName{Registry: "foo.com", Namespace: "spam", Repo: "bar", Tag: "1"}

		Expect(i1 == i2).To(BeTrue())
		Expect(i1 != i2).To(BeFalse())

		i2 = ImageName{Registry: "foo.com", Namespace: "spam", Repo: "bar", Tag: "2"}
		Expect(i1 == i2).To(BeFalse())
		Expect(i1 != i2).To(BeTrue())
	})
})
