package utils

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("lens", func() {
	var data map[string]interface{}
	BeforeEach(func() {
		data = map[string]interface{}{
			"a": "b",
			"c": []interface{}{
				map[string]interface{}{
					"d": 1,
				},
				map[string]interface{}{
					"d": 2,
				},
			},
		}
	})

	It("should work for maps", func() {
		myLens := Lens().M("a").Build()
		answer, err := myLens.Lookup(data)
		Expect(err).To(Succeed())
		Expect(answer).To(Equal("b"))

		myLens = Lens().M("nothere").Build()
		answer, err = myLens.Lookup(data)
		Expect(err).To(MatchError(ErrNotFound))

		myLens = Lens().M("a").M("deadend").Build()
		answer, err = myLens.Lookup(data)
		Expect(err).To(MatchError(ErrNotFound))
	})

	It("should work for lists", func() {
		myLens := Lens().M("c").L(0).M("d").Build()
		answer, err := myLens.Lookup(data)
		Expect(err).To(Succeed())
		Expect(answer).To(Equal(1))

		myLens = Lens().M("c").L(2).M("d").Build()
		answer, err = myLens.Lookup(data)
		Expect(err).To(MatchError(ErrNotFound))

		myLens = Lens().M("c").L(-1).M("d").Build()
		answer, err = myLens.Lookup(data)
		Expect(err).To(MatchError(ErrNotFound))

		myLens = Lens().M("c").L(0).L(0).Build()
		answer, err = myLens.Lookup(data)
		Expect(err).To(MatchError(ErrNotFound))

		myLens = Lens().M("a").Build()
		answer, err = myLens.L(data)
		Expect(err).To(HaveOccurred())
	})

	It("should work for collecting lists", func() {
		myLens := Lens().M("c").Apply(Lens().M("d").Build()).Build()
		answer, err := myLens.L(data)
		Expect(err).To(Succeed())
		Expect(answer).To(Equal([]interface{}{1, 2}))

		myLens = Lens().M("c").Apply(Lens().M("nothere").Build()).Build()
		answer, err = myLens.L(data)
		Expect(err).To(Succeed())
		Expect(answer).To(HaveLen(0))

		myLens = Lens().M("a").Apply(Lens().M("k").Build()).Build()
		answer, err = myLens.L(data)
		Expect(err).To(MatchError(ErrNotFound))

		myLens = Lens().M("c").Apply(Lens().M("d").L(0).Build()).Build()
		answer, err = myLens.L(data)
		Expect(err).To(Succeed())
		Expect(answer).To(HaveLen(0))

		myLens = Lens().M("c").Apply(Lens().M("d").L(0).Build()).Build()
		_, err = myLens.M(data)
		Expect(err).To(HaveOccurred())
	})
})
