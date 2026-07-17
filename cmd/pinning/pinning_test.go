package pinning

import (
	"bytes"
	"encoding/json"
	"html/template"
	"os"
	"path/filepath"

	. "github.com/benjamintf1/unmarshalledmatchers"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/operator-framework/operator-manifest-tools/internal/utils"
	"github.com/operator-framework/operator-manifest-tools/pkg/imageresolver"
	"go.yaml.in/yaml/v3"
)

const (
	testEggsImageRef = "registry.example.com/eggs:9.8"
	testSpamImageRef = "registry.example.com/maps/spam-operator:1.2"
)

var _ = Describe("pinning", func() {
	var (
		csvOriginal *template.Template
		// relatedImage,
		resolved                               *template.Template
		manifestDir, csvFilePath               string
		eggsImageReference, spamImageReference string

		resolver imageresolver.ImageResolver

		dir string
	)

	BeforeEach(func() {
		csvOriginal = template.Must(template.New("original").Parse(csvTemplate))
		resolved = template.Must(template.New("resolved").Parse(csvResolvedTemplate))

		dir, err := os.MkdirTemp("", "script")
		Expect(err).To(Succeed())
		DeferCleanup(func() {
			Expect(os.RemoveAll(dir)).To(Succeed())
		})

		manifestDir, err = os.MkdirTemp("", "pinning_test_")
		Expect(err).To(Succeed())
		DeferCleanup(func() {
			Expect(os.RemoveAll(manifestDir)).To(Succeed())
		})
		csvFilePath = filepath.Join(manifestDir, "clusterserviceversion.yaml")

		resolverScript := filepath.Join(dir, "resolver.sh")

		//nolint:gosec // test script needs execute permission
		err = os.WriteFile(resolverScript, []byte(`#!/bin/bash
if [ "$1" == "registry.example.com/eggs:9.8" ]; then
   echo -n "2"
   exit 0
fi

if [ "$1" == "registry.example.com/maps/spam-operator:1.2" ]; then
   echo -n "1"
   exit 0
fi

exit 1
`), 0o700)
		Expect(err).To(Succeed())

		resolver, _ = imageresolver.GetResolver(imageresolver.ResolverScript, map[string]string{
			"path": resolverScript,
		})
	})

	AfterEach(func() {
		// os.Remove(csvFilePath)
	})

	Context("extract", func() {
		BeforeEach(func() {
			eggsImageReference = testEggsImageRef
			spamImageReference = "registry.example.com/maps/spam-operator@sha256:1"

			csvFile, err := os.OpenFile(csvFilePath, os.O_CREATE|os.O_RDWR, 0o600) //nolint:gosec // test file path
			Expect(err).To(Succeed())
			defer func() { _ = csvFile.Close() }()

			Expect(csvOriginal.Execute(csvFile,
				struct {
					Vars map[string]string
				}{
					map[string]string{
						"Eggs": eggsImageReference,
						"Spam": spamImageReference,
					},
				})).To(Succeed())
		})

		It("should perform extract from csv", func() {
			extractData := bytes.Buffer{}
			Expect(extract(manifestDir, &extractData)).To(Succeed())

			extractJSON := []any{}

			Expect(json.Unmarshal(extractData.Bytes(), &extractJSON)).To(Succeed())
			Expect(extractJSON).To(HaveLen(2))
			Expect(extractJSON).To(ConsistOf(eggsImageReference, spamImageReference))
		})
	})

	Context("resolve", func() {
		var extractData []byte

		BeforeEach(func() {
			eggsImageReference = testEggsImageRef
			spamImageReference = testSpamImageRef

			csvFile, err := os.OpenFile(csvFilePath, os.O_CREATE|os.O_RDWR, 0o600) //nolint:gosec // test file path
			Expect(err).To(Succeed())
			defer func() { _ = csvFile.Close() }()

			Expect(csvOriginal.Execute(csvFile,
				struct {
					Vars map[string]string
				}{
					map[string]string{
						"Eggs": eggsImageReference,
						"Spam": spamImageReference,
					},
				})).To(Succeed())

			extractData, _ = json.Marshal([]any{
				testEggsImageRef,
				testSpamImageRef,
			})
		})

		It("should resolve image references", func() {
			resolveData := bytes.Buffer{}
			err := resolve(resolver, bytes.NewReader(extractData), &resolveData)
			Expect(err).To(Succeed())

			resolveJSON := map[string]any{}
			Expect(json.Unmarshal(resolveData.Bytes(), &resolveJSON)).To(Succeed())
			Expect(resolveJSON).To(HaveLen(2))
			Expect(resolveJSON).To(Equal(
				map[string]any{
					"registry.example.com/eggs:9.8":               "registry.example.com/eggs@sha256:2",
					"registry.example.com/maps/spam-operator:1.2": "registry.example.com/maps/spam-operator@sha256:1",
				}))
		})
	})

	Context("replace", func() {
		var (
			resolveData  []byte
			resolvedFile []byte
		)

		BeforeEach(func() {
			eggsImageReference = testEggsImageRef
			spamImageReference = testSpamImageRef

			csvFile, err := os.OpenFile(csvFilePath, os.O_CREATE|os.O_WRONLY, 0o600) //nolint:gosec // test file path
			Expect(err).To(Succeed())
			defer func() { _ = csvFile.Close() }()

			Expect(csvOriginal.Execute(csvFile,
				struct {
					Vars map[string]string
				}{
					map[string]string{
						"Eggs": eggsImageReference,
						"Spam": spamImageReference,
					},
				})).To(Succeed())

			var resolvedFileBuffer bytes.Buffer
			Expect(resolved.Execute(&resolvedFileBuffer,
				struct {
					Vars map[string]string
				}{
					map[string]string{
						"Eggs": "registry.example.com/eggs@sha256:2",
						"Spam": "registry.example.com/maps/spam-operator@sha256:1",
					},
				})).To(Succeed())

			resolvedFile = resolvedFileBuffer.Bytes()

			resolveData, _ = json.Marshal(map[string]any{
				"registry.example.com/eggs:9.8":               "registry.example.com/eggs@sha256:2",
				"registry.example.com/maps/spam-operator:1.2": "registry.example.com/maps/spam-operator@sha256:1",
			})
		})

		It("should replace image refs", func() {
			err := replace(manifestDir, bytes.NewReader(resolveData))
			Expect(err).To(Succeed())

			fileData, err := os.ReadFile(csvFilePath) //nolint:gosec // test file path
			Expect(err).To(Succeed())

			Expect(fileData).To(MatchUnorderedYAML(resolvedFile))
		})
	})

	Context("pin", func() {
		var (
			outputExtract, outputReplace utils.OutputParam

			resolvedFile             []byte
			extractFile, replaceFile *os.File
		)

		BeforeEach(func() {
			eggsImageReference = testEggsImageRef
			spamImageReference = testSpamImageRef

			csvFile, err := os.OpenFile(csvFilePath, os.O_CREATE|os.O_WRONLY, 0o600) //nolint:gosec // test file path
			Expect(err).To(Succeed())
			defer func() { _ = csvFile.Close() }()

			Expect(csvOriginal.Execute(csvFile,
				struct {
					Vars map[string]string
				}{
					map[string]string{
						"Eggs": eggsImageReference,
						"Spam": spamImageReference,
					},
				})).To(Succeed())

			extractFile, err = os.CreateTemp(dir, "extract")
			Expect(err).To(Succeed())
			outputExtract = utils.NewOutputParam()
			outputExtract.Name = extractFile.Name()

			replaceFile, err = os.CreateTemp(dir, "replace")
			Expect(err).To(Succeed())
			outputReplace = utils.NewOutputParam()
			outputReplace.Name = replaceFile.Name()

			var resolvedFileBuffer bytes.Buffer
			Expect(resolved.Execute(&resolvedFileBuffer,
				struct {
					Vars map[string]string
				}{
					map[string]string{
						"Eggs": "registry.example.com/eggs@sha256:2",
						"Spam": "registry.example.com/maps/spam-operator@sha256:1",
					},
				})).To(Succeed())

			resolvedFile = resolvedFileBuffer.Bytes()
		})

		AfterEach(func() {
			_ = os.Remove(extractFile.Name())
			_ = os.Remove(replaceFile.Name())
		})

		It("should replace image refs", func() {
			err := pin(
				manifestDir,
				resolver,
				outputExtract,
				outputReplace,
			)
			Expect(err).To(Succeed())

			extractAnswer, err := os.ReadFile(outputExtract.Name)
			Expect(err).To(Succeed())

			extractJSON := []any{}

			Expect(json.Unmarshal(extractAnswer, &extractJSON)).To(Succeed())
			Expect(extractJSON).To(HaveLen(2))
			Expect(extractJSON).To(ConsistOf(eggsImageReference, spamImageReference))
			Expect(err).To(Succeed())

			resolveAnswer, err := os.ReadFile(outputReplace.Name)
			Expect(err).To(Succeed())

			resolveJSON := map[string]any{}
			Expect(json.Unmarshal(resolveAnswer, &resolveJSON)).To(Succeed())
			Expect(resolveJSON).To(HaveLen(2))
			Expect(resolveJSON).To(Equal(
				map[string]any{
					"registry.example.com/eggs:9.8":               "registry.example.com/eggs@sha256:2",
					"registry.example.com/maps/spam-operator:1.2": "registry.example.com/maps/spam-operator@sha256:1",
				}))

			replaceAnswer, err := os.ReadFile(csvFilePath) //nolint:gosec // test file path
			Expect(err).To(Succeed())
			Expect(replaceAnswer).To(MatchUnorderedYAML(resolvedFile))

			validYaml := map[string]any{}
			Expect(yaml.Unmarshal(replaceAnswer, &validYaml)).To(Succeed())
		})
	})
})

const csvTemplate = `apiVersion: operators.coreos.com/v1alpha1
kind: ClusterServiceVersion
metadata:
  name: foo
spec:
  install:
    spec:
      deployments:
      - spec:
          template:
            spec:
              containers:
              - name: spam-operator
                image: {{.Vars.Spam}}
              - name: eggs
                image: {{.Vars.Eggs}}
`

const csvResolvedTemplate = `apiVersion: operators.coreos.com/v1alpha1
kind: ClusterServiceVersion
metadata:
  name: foo
spec:
  install:
    spec:
      deployments:
      - spec:
          template:
            spec:
              containers:
              - name: spam-operator
                image: {{.Vars.Spam}}
              - name: eggs
                image: {{.Vars.Eggs}}
  relatedImages:
  - name: eggs
    image: {{.Vars.Eggs}}
  - name: spam-operator
    image: {{.Vars.Spam}}
`
