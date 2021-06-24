package pullspec

import (
	"fmt"

	. "github.com/onsi/ginkgo/extensions/table"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const sha = "5d141ae1081640587636880dbe8489439353df883379158fa8742d5a3be75475"
const notB16 = "5d141ae1081640587636880dbe8489439353df883379158fa8742d5a3be7547g"

var _ = Describe("DefaultPullspecHeuristic", func() {
	DescribeTable("matches",
		func(text string, expected []string) {
			result := DefaultPullspecHeuristic(text)
			strs := []string{}

			for _, bounds := range result {
				strs = append(strs, text[bounds[0]:bounds[1]])
			}

			Expect(strs).To(ConsistOf(expected))
		},
		// Trivial cases
    Entry("trivial case", "a.b/c:1", []string{"a.b/c:1"}),
    Entry("trivial case", "a.b/c/d:1", []string{"a.b/c/d:1"}),
    Entry("trivial case", "registry.i/namespace/baz:latest", []string{"registry.i/namespace/baz:latest"}),
    // Digests in tag
    Entry("digests", fmt.Sprintf("a.b/c@sha256:%s", sha), []string{fmt.Sprintf("a.b/c@sha256:%s", sha)}),
    Entry("digests", fmt.Sprintf("a.b/c/d@sha256:%s", sha), []string{fmt.Sprintf("a.b/c/d@sha256:%s", sha)}),
    // Port in registry
    Entry("", "a.b:1/c:1", []string{"a.b:1/c:1"}),
    Entry("", "a.b:5000/c/d:1", []string{"a.b:5000/c/d:1"}),
    // Special characters everywhere
    Entry("", fmt.Sprintf("a-b.c_d/e-f.g_h/i-j.k_l@sha256:%s", sha),
			[]string{fmt.Sprintf("a-b.c_d/e-f.g_h/i-j.k_l@sha256:%s", sha)}),
    Entry("", "a-._b/c-._d/e-._f:g-._h", []string{"a-._b/c-._d/e-._f:g-._h"}),
    Entry("", "1.2-3_4/5.6-7_8/9.0-1_2:3.4-5_6", []string{"1.2-3_4/5.6-7_8/9.0-1_2:3.4-5_6"}),
    // Multiple namespaces
    Entry("", "a.b/c/d/e:1", []string{"a.b/c/d/e:1"}),
    Entry("", "a.b/c/d/e/f/g/h/i:1", []string{"a.b/c/d/e/f/g/h/i:1"}),
    // Enclosed in various non-pullspec characters
    Entry("", " a.b/c:1 ", []string{"a.b/c:1"}),
    Entry("", "\na.b/c:1\n", []string{"a.b/c:1"}),
    Entry("", "\ta.b/c:1\t", []string{"a.b/c:1"}),
    Entry("", ",a.b/c:1,", []string{"a.b/c:1"}),
    Entry("", ";a.b/c:1;", []string{"a.b/c:1"}),
    Entry("", "'a.b/c:1'", []string{"a.b/c:1"}),
    Entry("", `"a.b/c:1"`, []string{"a.b/c:1"}),
    Entry("", "<a.b/c:1>", []string{"a.b/c:1"}),
    Entry("", "`a.b/c:1`", []string{"a.b/c:1"}),
    Entry("", "*a.b/c:1*", []string{"a.b/c:1"}),
    Entry("", "(a.b/c:1)", []string{"a.b/c:1"}),
    Entry("", "[a.b/c:1]", []string{"a.b/c:1"}),
    Entry("", "{a.b/c:1}", []string{"a.b/c:1"}),
    // Enclosed in various pullspec characters
    Entry("", ".a.b/c:1.", []string{"a.b/c:1"}),
    Entry("", "-a.b/c:1-", []string{"a.b/c:1"}),
    Entry("", "_a.b/c:1_", []string{"a.b/c:1"}),
    Entry("", "/a.b/c:1/", []string{"a.b/c:1"}),
    Entry("", "@a.b/c:1@", []string{"a.b/c:1"}),
    Entry("", ":a.b/c:1:", []string{"a.b/c:1"}),
    // Enclosed in multiple pullspec characters
    Entry("", "...a.b/c:1...", []string{"a.b/c:1"}),
    // Redundant but important interaction of ^ with tags
    Entry("", "a.b/c:latest:", []string{"a.b/c:latest"}),
    Entry("", fmt.Sprintf("a.b/c@sha256:%s:", sha), []string{fmt.Sprintf("a.b/c@sha256:%s", sha)}),
    Entry("", fmt.Sprintf("a.b/c@sha256:%s...", sha), []string{fmt.Sprintf("a.b/c@sha256:%s", sha)}),
    Entry("", "a.b/c:v1.1...", []string{"a.b/c:v1.1"}),
    // Empty-ish strings
    Entry("", "", []string{}),
    Entry("", "!", []string{}),
    Entry("", ".", []string{}),
    Entry("", "!!!", []string{}),
    Entry("", "...", []string{}),
    // Not enough parts
    Entry("", "a.bc:1", []string{}),
    // No '.' in registry
    Entry("", "ab/c:1", []string{}),
    // No tag
    Entry("", "a.b/c", []string{}),
    Entry("", "a.b/c:", []string{}),
    Entry("", "a.b/c:...", []string{}),
    // Invalid digest
    Entry("", "a.b/c:@123", []string{}),
    Entry("", "a.b/c:@:123", []string{}),
    Entry("", "a.b/c:@sha256", []string{}),
    Entry("", "a.b/c:@sha256:", []string{}),
    Entry("", "a.b/c:@sha256:...", []string{}),
    Entry("", "a.b/c:@sha256:123456", []string{}),   // Must be 64 characters
    Entry("", fmt.Sprintf("a.b/c:@sha256:%s", notB16), []string{}),
    // Empty part
    Entry("", "a.b//c:1", []string{}),
    Entry("", "https://a.b/c:1", []string{}),
    // '@' in registry
    Entry("", "a@b.c/d:1", []string{}),
    Entry("", "a.b@c/d:1", []string{}),
    // '@' or ':' in namespace
    Entry("", "a.b/c@d/e:1", []string{}),
    Entry("", "a.b/c:d/e:1", []string{}),
    Entry("", "a.b/c/d@e/f:1", []string{}),
    Entry("", "a.b/c/d:e/f:1", []string{}),
    // Invalid port in registry
    Entry("", "a:b.c/d:1", []string{}),
    Entry("", "a.b:c/d:1", []string{}),
    Entry("", "a.b:/c:1", []string{}),
    Entry("", "a.b:11ff/c:1", []string{}),
    // Some part does not start/end with an alphanumeric character
    Entry("", "a.b-/c:1", []string{}),
    Entry("", "a.b/-c:1", []string{}),
    Entry("", "a.b/c-:1", []string{}),
    Entry("", "a.b/c:-1", []string{}),
    Entry("", "a.b/-c/d:1", []string{}),
    Entry("", "a.b/c-/d:1", []string{}),
    Entry("", "a.b/c/-d:1", []string{}),
    Entry("", "a.b/c/d-:1", []string{}),
    Entry("", "a.b/c/d:-1", []string{}),
    // Separated by various non-pullspec characters
    Entry("", "a.b/c:1 d.e/f:1", []string{"a.b/c:1", "d.e/f:1"}),
    Entry("", "a.b/c:1\td.e/f:1", []string{"a.b/c:1", "d.e/f:1"}),
    Entry("", "a.b/c:1\nd.e/f:1", []string{"a.b/c:1", "d.e/f:1"}),
    Entry("", "a.b/c:1\n\t d.e/f:1", []string{"a.b/c:1", "d.e/f:1"}),
    Entry("", "a.b/c:1,d.e/f:1", []string{"a.b/c:1", "d.e/f:1"}),
    Entry("", "a.b/c:1;d.e/f:1", []string{"a.b/c:1", "d.e/f:1"}),
    Entry("", "a.b/c:1, d.e/f:1", []string{"a.b/c:1", "d.e/f:1"}),
    Entry("", "a.b/c:1; d.e/f:1", []string{"a.b/c:1", "d.e/f:1"}),
    Entry("", "a.b/c:1 , d.e/f:1", []string{"a.b/c:1", "d.e/f:1"}),
    Entry("", "a.b/c:1 ; d.e/f:1", []string{"a.b/c:1", "d.e/f:1"}),
    // Separated by pullspec characters
    // Note the space on at least one side of the separator, will not work otherwise
    Entry("", "a.b/c:1/ d.e/f:1", []string{"a.b/c:1", "d.e/f:1"}),
    Entry("", "a.b/c:1 /d.e/f:1", []string{"a.b/c:1", "d.e/f:1"}),
    Entry("", "a.b/c:1- d.e/f:1", []string{"a.b/c:1", "d.e/f:1"}),
    Entry("", "a.b/c:1 -d.e/f:1", []string{"a.b/c:1", "d.e/f:1"}),
    Entry("", "a.b/c:1: d.e/f:1", []string{"a.b/c:1", "d.e/f:1"}),
    Entry("", "a.b/c:1 :d.e/f:1", []string{"a.b/c:1", "d.e/f:1"}),
    Entry("", "a.b/c:1. d.e/f:1", []string{"a.b/c:1", "d.e/f:1"}),
    Entry("", "a.b/c:1 .d.e/f:1", []string{"a.b/c:1", "d.e/f:1"}),
    Entry("", "a.b/c:1_ d.e/f:1", []string{"a.b/c:1", "d.e/f:1"}),
    Entry("", "a.b/c:1 _d.e/f:1", []string{"a.b/c:1", "d.e/f:1"}),
    Entry("", "a.b/c:1@ d.e/f:1", []string{"a.b/c:1", "d.e/f:1"}),
    Entry("", "a.b/c:1 @d.e/f:1", []string{"a.b/c:1", "d.e/f:1"}),
    // Sentences
    Entry("", "First is a.b/c:1. Second is d.e/f:1.", []string{"a.b/c:1", "d.e/f:1"}),
    Entry("", "My pullspecs are a.b/c:1 and d.e/f:1.", []string{"a.b/c:1", "d.e/f:1"}),
    Entry("", "There is/are some pullspec(s) in registry.io: a.b/c:1, d.e/f:1", []string{"a.b/c:1", "d.e/f:1"}),
    Entry("", `
     Find more info on https://my-site.com/here.
     Some pullspec are <i>a.b/c:1<i> and __d.e/f:1__.
     There is also g.h/i:latest: that one is cool.
     And you can email me at name@server.com for info
     about the last one: j.k/l:v1.1.
     `, []string{"a.b/c:1", "d.e/f:1", "g.h/i:latest", "j.k/l:v1.1"}),
    Entry("", `
     I might also decide to do some math: 50.0/2 = 25.0.
     Perhaps even with variables: 0.5x/2 = x/4.
     And, because I am a psychopath, I will write this: 0.5/2:2 = 1/8,
     Which will be a false positive.
     `, []string{"0.5/2:2"}),
    // JSON/YAML strings
    Entry("", `[]string{"a.b/c:1","d.e/f:1", "g.h/i:1"]`, []string{"a.b/c:1", "d.e/f:1", "g.h/i:1"}),
    Entry("", `{"a":"a.b/c:1","b": "d.e/f:1", "c": "g.h/i:1"}`, []string{"a.b/c:1", "d.e/f:1", "g.h/i:1"}),
    Entry("", "[a.b/c:1,d.e/f:1, g.h/i:1]", []string{"a.b/c:1", "d.e/f:1", "g.h/i:1"}),
    Entry("", "{a: a.b/c:1,b: d.e/f:1, c: g.h/i:1}", []string{"a.b/c:1", "d.e/f:1", "g.h/i:1"}),
    Entry("", `
     a: a.b/c:1
     b: d.e/f:1
     c: g.h/i:1
     `, []string{"a.b/c:1", "d.e/f:1", "g.h/i:1"}),
	)
})
