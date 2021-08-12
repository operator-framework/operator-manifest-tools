package pullspec

import (
	"strings"

	"github.com/operator-framework/operator-manifest-tools/pkg/imagename"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"k8s.io/apimachinery/pkg/util/intstr"

	"text/template"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("OperatorCSV", func() {
	var pullSpecMap map[string]pullSpec
	var original, replaced, replacedEverywhere csvFile
	var originalPullSpecs, replacementPullSpecs []*imagename.ImageName
	var replacements map[imagename.ImageName]imagename.ImageName
	var dec runtime.Serializer

	BeforeEach(func() {
		pullSpecMap = make(map[string]pullSpec)

		originalPullSpecs = []*imagename.ImageName{}
		replacementPullSpecs = []*imagename.ImageName{}
		replacements = make(map[imagename.ImageName]imagename.ImageName)

		for i := range pullSpecs {
			pullSpecMap[pullSpecs[i].Name] = pullSpecs[i]

			originalPullSpecs = append(originalPullSpecs, pullSpecs[i].value)
			replacementPullSpecs = append(replacementPullSpecs, pullSpecs[i].replace)
			replacements[*pullSpecs[i].value] = *pullSpecs[i].replace
		}

		strb := strings.Builder{}
		Expect(template.Must(template.New("original").Delims("{", "}").Parse(originalContent)).Execute(&strb, pullSpecMap)).To(Succeed())

		data := &unstructured.Unstructured{}
		dec = yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
		_, _, err := dec.Decode([]byte(strb.String()), nil, data)
		Expect(err).To(Succeed())
		original = csvFile{data: data}

		strb = strings.Builder{}
		Expect(template.Must(template.New("replaced").Delims("{", "}").Parse(replacedContent)).Execute(&strb, pullSpecMap)).To(Succeed())

		data = &unstructured.Unstructured{}
		_, _, err = dec.Decode([]byte(strb.String()), nil, data)
		Expect(err).To(Succeed())
		replaced = csvFile{data: data}

		strb = strings.Builder{}
		Expect(template.Must(template.New("replacedEverywhere").Delims("{", "}").Parse(replacedEverywhereContent)).Execute(&strb, pullSpecMap)).To(Succeed())

		data = &unstructured.Unstructured{}
		_, _, err = dec.Decode([]byte(strb.String()), nil, data)
		Expect(err).To(Succeed())
		replacedEverywhere = csvFile{data: data}
	})

	It("should fail on wrong kind", func() {
		original.data.SetKind("")
		_, err := NewOperatorCSV("original.yaml", original.data, nil)
		Expect(err).To(MatchError(ErrNotClusterServiceVersion))
	})

	It("should fail on wrong kind name", func() {
		original.data.SetKind("ClusterResourceDefinition")
		_, err := NewOperatorCSV("original.yaml", original.data, nil)
		Expect(err).To(MatchError(ErrNotClusterServiceVersion))
	})

	It("should get pullspecs", func() {
		csv, err := NewOperatorCSV("original.yaml", original.data, nil)
		Expect(err).To(Succeed())

		pullSpecs, err := csv.GetPullSpecs()
		Expect(err).To(Succeed())
		Expect(pullSpecs).NotTo(HaveLen(0))
		Expect(pullSpecs).To(ConsistOf(originalPullSpecs))
	})

	It("should replace pullspecs", func() {
		csv, err := NewOperatorCSV("original.yaml", original.data, nil)
		Expect(err).To(Succeed())
		Expect(csv.ReplacePullSpecs(replacements)).To(Succeed())

		strb := strings.Builder{}
		Expect(dec.Encode(&csv.data, &strb)).To(Succeed())
		dataStr := strb.String()

		Expect(dec.Encode(replaced.data, &strb)).To(Succeed())
		replacedStr := strb.String()
		Expect(dataStr).To(MatchYAML(replacedStr))
	})

	It("should replace pullspecs everywhere", func() {
		csv, err := NewOperatorCSV("original.yaml", original.data, nil)
		Expect(err).To(Succeed())
		Expect(csv.ReplacePullSpecsEverywhere(replacements)).To(Succeed())

		strb := strings.Builder{}
		Expect(dec.Encode(&csv.data, &strb)).To(Succeed())
		dataStr := strb.String()

		Expect(dec.Encode(replacedEverywhere.data, &strb)).To(Succeed())
		replacedStr := strb.String()
		Expect(dataStr).To(MatchYAML(replacedStr))
	})
})

type csvFile struct {
	data *unstructured.Unstructured
}

type pullSpec struct {
	Name           string
	value, replace *imagename.ImageName
	Path           []intstr.IntOrString
}

func (s pullSpec) String() string {
	return s.value.String()
}

func (s pullSpec) Replace() string {
	return s.replace.String()
}

func newPullSpec(name, value, replace string, path []string) pullSpec {
	parsedPath := []intstr.IntOrString{}

	for i := range path {
		parsedPath = append(parsedPath, intstr.FromString(path[i]))
	}

	valueImg := imagename.Parse(value)
	replaced := imagename.Parse(replace)

	return pullSpec{
		Name:    name,
		value:   valueImg,
		replace: replaced,
		Path:    parsedPath,
	}
}

var (
	RI1 = newPullSpec(
		"ri1", "foo:1", "r-foo:2",
		[]string{"spec", "relatedImages", "0", "image"},
	)
	RI2 = newPullSpec(
		"ri2", "registry/bar:1", "r-registry/r-bar:2",
		[]string{"spec", "relatedImages", "1", "image"},
	)
	C1 = newPullSpec(
		"c1", "registry/namespace/spam:1", "r-registry/r-namespace/r-spam:2",
		[]string{"spec", "install", "spec", "deployments", "0",
			"spec", "template", "spec", "containers", "0", "image"},
	)
	CE1 = newPullSpec(
		"ce1", "eggs:1", "r-eggs:2",
		[]string{"spec", "install", "spec", "deployments", "0",
			"spec", "template", "spec", "containers", "0", "env", "0", "value"},
	)
	C2 = newPullSpec(
		"c2", "ham:1", "r-ham:2",
		[]string{"spec", "install", "spec", "deployments", "0",
			"spec", "template", "spec", "containers", "1", "image"},
	)
	C3 = newPullSpec(
		"c3", "jam:1", "r-jam:2",
		[]string{"spec", "install", "spec", "deployments", "1",
			"spec", "template", "spec", "containers", "0", "image"},
	)
	AN1 = newPullSpec(
		"an1", "registry.io/namespace/baz:latest", "r-registry.io/r-namespace/r-baz:latest",
		[]string{"metadata", "annotations", "containerImage"},
	)
	IC1 = newPullSpec(
		"ic1", "pullspec:1", "r-pullspec:1",
		[]string{"spec", "install", "spec", "deployments", "1",
			"spec", "template", "spec", "initContainers", "0", "image"},
	)
	ICE1 = newPullSpec(
		"ice1", "pullspec:2", "r-pullspec:2",
		[]string{"spec", "install", "spec", "deployments", "1",
			"spec", "template", "spec", "initContainers", "0", "env", "0", "value"},
	)
	AN2 = newPullSpec(
		"an2", "registry.io/an2:1", "registry.io/r-an2:1",
		[]string{"metadata", "annotations", "some_pullspec"},
	)
	AN3 = newPullSpec(
		"an3", "registry.io/an3:1", "registry.io/r-an3:1",
		[]string{"metadata", "annotations", "two_pullspecs"},
	)
	AN4 = newPullSpec(
		"an4", "registry.io/an4:1", "registry.io/r-an4:1",
		[]string{"metadata", "annotations", "two_pullspecs"},
	)
	AN5 = newPullSpec(
		"an5", "registry.io/an5:1", "registry.io/r-an5:1",
		[]string{"spec", "install", "spec", "deployments", "0",
			"spec", "template", "metadata", "annotations", "some_other_pullspec"},
	)
	AN6 = newPullSpec(
		"an6", "registry.io/an6:1", "registry.io/r-an6:1",
		[]string{"random", "annotations", "0", "metadata", "annotations", "duplicate_pullspecs"},
	)
	AN7 = newPullSpec(
		"an7", "registry.io/an7:1", "registry.io/r-an7:1",
		[]string{"random", "annotations", "0", "metadata", "annotations", "duplicate_pullspecs"},
	)
	pullSpecs = []pullSpec{
		RI1, RI2, C1, CE1, C2, C3, AN1, IC1, ICE1, AN2, AN3, AN4, AN5, AN6, AN7,
	}
)

const originalContent = `
# A meaningful comment
kind: ClusterServiceVersion
metadata:
  annotations:
    containerImage: {.an1}
    some_pullspec: {.an2}
    two_pullspecs: {.an3}, {.an4}
spec:
  relatedImages:
  - name: ri1
    image: {.ri1}
  - name: ri2
    image: {.ri2}
  install:
    spec:
      deployments:
      - spec:
          template:
            metadata:
              annotations:
                some_other_pullspec: {.an5}
            spec:
              containers:
              - name: c1
                image: {.c1}
                env:
                - name: RELATED_IMAGE_CE1
                  value: {.ce1}
                - name: UNRELATED_IMAGE
                  value: {.ce1}
                - name: UNRELATED_ENV_VAR
                  valueFrom:
                    fieldRef:
                      fieldPath: metadata.namespace
              - name: c2
                image: {.c2}
      - spec:
          template:
            spec:
              containers:
              - name: c3
                image: {.c3}
              initContainers:
              - name: ic1
                image: {.ic1}
                env:
                - name: RELATED_IMAGE_ICE1
                  value: {.ice1}
random:
  annotations:
  - metadata:
      annotations:
        duplicate_pullspecs: {.an6}, {.an7}, {.an6}, {.an7}
  nested:
    dict:
      a: {.ri1}
      b: {.ri2}
      c: {.c1}
      d: {.ce1}
      e: {.c2}
      f: {.c3}
      g: {.an1}
      h: {.ic1}
      i: {.ice1}
    list:
    - {.ri1}
    - {.ri2}
    - {.c1}
    - {.ce1}
    - {.c2}
    - {.c3}
    - {.an1}
    - {.ic1}
    - {.ice1}
`

const replacedContent = `
# A meaningful comment
kind: ClusterServiceVersion
metadata:
  annotations:
    containerImage: {.an1.Replace}
    some_pullspec: {.an2.Replace}
    two_pullspecs: {.an3.Replace}, {.an4.Replace}
spec:
  relatedImages:
  - name: ri1
    image: {.ri1.Replace}
  - name: ri2
    image: {.ri2.Replace}
  install:
    spec:
      deployments:
      - spec:
          template:
            metadata:
              annotations:
                some_other_pullspec: {.an5.Replace}
            spec:
              containers:
              - name: c1
                image: {.c1.Replace}
                env:
                - name: RELATED_IMAGE_CE1
                  value: {.ce1.Replace}
                - name: UNRELATED_IMAGE
                  value: {.ce1}
                - name: UNRELATED_ENV_VAR
                  valueFrom:
                    fieldRef:
                      fieldPath: metadata.namespace
              - name: c2
                image: {.c2.Replace}
      - spec:
          template:
            spec:
              containers:
              - name: c3
                image: {.c3.Replace}
              initContainers:
              - name: ic1
                image: {.ic1.Replace}
                env:
                - name: RELATED_IMAGE_ICE1
                  value: {.ice1.Replace}
random:
  annotations:
  - metadata:
      annotations:
        duplicate_pullspecs: {.an6.Replace}, {.an7.Replace}, {.an6.Replace}, {.an7.Replace}
  nested:
    dict:
      a: {.ri1}
      b: {.ri2}
      c: {.c1}
      d: {.ce1}
      e: {.c2}
      f: {.c3}
      g: {.an1}
      h: {.ic1}
      i: {.ice1}
    list:
    - {.ri1}
    - {.ri2}
    - {.c1}
    - {.ce1}
    - {.c2}
    - {.c3}
    - {.an1}
    - {.ic1}
    - {.ice1}
`

const replacedEverywhereContent = `
# A meaningful comment
kind: ClusterServiceVersion
metadata:
  annotations:
    containerImage: {.an1.Replace}
    some_pullspec: {.an2.Replace}
    two_pullspecs: {.an3.Replace}, {.an4.Replace}
spec:
  relatedImages:
  - name: ri1
    image: {.ri1.Replace}
  - name: ri2
    image: {.ri2.Replace}
  install:
    spec:
      deployments:
      - spec:
          template:
            metadata:
              annotations:
                some_other_pullspec: {.an5.Replace}
            spec:
              containers:
              - name: c1
                image: {.c1.Replace}
                env:
                - name: RELATED_IMAGE_CE1
                  value: {.ce1.Replace}
                - name: UNRELATED_IMAGE
                  value: {.ce1.Replace}
                - name: UNRELATED_ENV_VAR
                  valueFrom:
                    fieldRef:
                      fieldPath: metadata.namespace
              - name: c2
                image: {.c2.Replace}
      - spec:
          template:
            spec:
              containers:
              - name: c3
                image: {.c3.Replace}
              initContainers:
              - name: ic1
                image: {.ic1.Replace}
                env:
                - name: RELATED_IMAGE_ICE1
                  value: {.ice1.Replace}
random:
  annotations:
  - metadata:
      annotations:
        duplicate_pullspecs: {.an6.Replace}, {.an7.Replace}, {.an6.Replace}, {.an7.Replace}
  nested:
    dict:
      a: {.ri1.Replace}
      b: {.ri2.Replace}
      c: {.c1.Replace}
      d: {.ce1.Replace}
      e: {.c2.Replace}
      f: {.c3.Replace}
      g: {.an1.Replace}
      h: {.ic1.Replace}
      i: {.ice1.Replace}
    list:
    - {.ri1.Replace}
    - {.ri2.Replace}
    - {.c1.Replace}
    - {.ce1.Replace}
    - {.c2.Replace}
    - {.c3.Replace}
    - {.an1.Replace}
    - {.ic1.Replace}
    - {.ice1.Replace}
`
