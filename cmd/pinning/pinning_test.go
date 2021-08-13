package pinning

import (
	"bytes"
	"encoding/json"
	"html/template"
	"io/ioutil"
	"os"
	"path/filepath"

	. "github.com/benjamintf1/unmarshalledmatchers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/operator-framework/operator-manifest-tools/pkg/imageresolver"
)

var _ = Describe("pinning", func() {
	var (
		csvOriginal *template.Template
		//relatedImage,
		resolved    *template.Template
		manifestDir, csvFilePath                   string
		eggsImageReference, spamImageReference string

		resolver imageresolver.ImageResolver
	)

	BeforeEach(func() {
		csvOriginal = template.Must(template.New("original").Parse(CSV_TEMPLATE))
		// relatedImage = template.Must(template.New("relatedImage").
		// 	Delims("{", "}").Parse(CSV_TEMPLATE_WITH_RELATED_IMAGES))
		resolved = template.Must(template.New("resolved").Parse(CSV_RESOLVED_TEMPLATE))

		dir, _ := ioutil.TempDir("", "script")
		manifestDir, _ = ioutil.TempDir("", "pinning_test_")
		csvFilePath = filepath.Join(manifestDir, "clusterserviceversion.yaml")

		resolverScript := filepath.Join(dir, "resolver.sh")

		ioutil.WriteFile(resolverScript, []byte(`#!/bin/bash
if [ "$1" == "registry.example.com/eggs:9.8" ]; then
   echo -n "2"
   exit 0
fi

if [ "$1" == "registry.example.com/maps/spam-operator:1.2" ]; then
   echo -n "1"
   exit 0
fi

exit 1
`), 0700)

		resolver, _ = imageresolver.GetResolver(imageresolver.ResolverScript, map[string]string{
			"path": resolverScript,
		})
	})

	AfterEach(func() {
		//os.Remove(csvFilePath)
	})

	Context("extract", func() {
		BeforeEach(func() {
			eggsImageReference = "registry.example.com/eggs:9.8"
			spamImageReference = "registry.example.com/maps/spam-operator@sha256:1"

			csvFile, err := os.OpenFile(csvFilePath, os.O_CREATE|os.O_RDWR, 0755)
			defer csvFile.Close()

			Expect(err).To(Succeed())

			csvOriginal.Execute(csvFile,
				struct {
					Vars map[string]string
				}{
					map[string]string{
						"Eggs": eggsImageReference,
						"Spam": spamImageReference,
					},
				})
		})

		It("should perform extract from csv", func() {
			extractData := bytes.Buffer{}
			extract(manifestDir, &extractData)

			extractJson := []interface{}{}

			Expect(json.Unmarshal(extractData.Bytes(), &extractJson)).To(Succeed())
			Expect(extractJson).To(HaveLen(2))
			Expect(extractJson).To(ConsistOf(eggsImageReference, spamImageReference))
		})
	})

	Context("resolve", func() {
		var extractData []byte

		BeforeEach(func() {
			eggsImageReference = "registry.example.com/eggs:9.8"
			spamImageReference = "registry.example.com/maps/spam-operator:1.2"

			csvFile, err := os.OpenFile(csvFilePath, os.O_CREATE|os.O_RDWR, 0755)
			defer csvFile.Close()
			Expect(err).To(Succeed())

			csvOriginal.Execute(csvFile,
				struct {
					Vars map[string]string
				}{
					map[string]string{
						"Eggs": eggsImageReference,
						"Spam": spamImageReference,
					},
				})

			extractData, _ = json.Marshal([]interface{}{
				"registry.example.com/eggs:9.8",
				"registry.example.com/maps/spam-operator:1.2",
			})
		})

		It("should resolve image references", func() {
			resolveData := bytes.Buffer{}
			err := resolve(resolver, bytes.NewReader(extractData), &resolveData)
			Expect(err).To(Succeed())

			resolveJson := map[string]interface{}{}
			Expect(json.Unmarshal(resolveData.Bytes(), &resolveJson)).To(Succeed())
			Expect(resolveJson).To(HaveLen(2))
			Expect(resolveJson).To(Equal(
				map[string]interface{}{
					"registry.example.com/eggs:9.8":               "registry.example.com/eggs@sha256:2",
					"registry.example.com/maps/spam-operator:1.2": "registry.example.com/maps/spam-operator@sha256:1",
				}))
		})
	})

	Context("replace", func() {
		var (
			resolveData []byte
			resolvedFile []byte
		)

		BeforeEach(func() {
			eggsImageReference = "registry.example.com/eggs:9.8"
			spamImageReference = "registry.example.com/maps/spam-operator:1.2"

			csvFile, err := os.OpenFile(csvFilePath, os.O_CREATE|os.O_WRONLY, 0755)
			defer csvFile.Close()
			Expect(err).To(Succeed())

			csvOriginal.Execute(csvFile,
				struct {
					Vars map[string]string
				}{
					map[string]string{
						"Eggs": eggsImageReference,
						"Spam": spamImageReference,
					},
				})

			var resolvedFileBuffer bytes.Buffer
			resolved.Execute(&resolvedFileBuffer,
				struct {
					Vars map[string]string
				}{
					map[string]string{
						"Eggs": "registry.example.com/eggs@sha256:2",
						"Spam": "registry.example.com/maps/spam-operator@sha256:1",
					},
				})

			resolvedFile = resolvedFileBuffer.Bytes()

			resolveData, _ = json.Marshal(map[string]interface{}{
				"registry.example.com/eggs:9.8":               "registry.example.com/eggs@sha256:2",
				"registry.example.com/maps/spam-operator:1.2": "registry.example.com/maps/spam-operator@sha256:1",
			})
		})

		It("should replace image refs", func() {
			err := replace(manifestDir, bytes.NewReader(resolveData))
			Expect(err).To(Succeed())

			fileData, err := ioutil.ReadFile(csvFilePath)
			Expect(err).To(Succeed())

			Expect(fileData).To(MatchUnorderedYAML(resolvedFile))
		})
	})
})

const CSV_TEMPLATE = `apiVersion: operators.coreos.com/v1alpha1
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

const CSV_TEMPLATE_WITH_RELATED_IMAGES = CSV_TEMPLATE + `
  relatedImages:
    image: {{.Vars.Spam}}
  - name: eggs
    image: {{.Vars.Eggs}}
`

const CSV_RESOLVED_TEMPLATE = `apiVersion: operators.coreos.com/v1alpha1
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
