package pullspec

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"io/fs"

	"github.com/operator-framework/operator-manifest-tools/pkg/imagename"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
)

type NamedPullSpec interface {
	fmt.Stringer
	Name() string
	Image() string
	Data() map[string]interface{}
	SetImage(string)
	AsYamlObject() map[string]interface{}
}

type namedPullSpec struct {
	imageKey string
	data     map[string]interface{}
}

func (named *namedPullSpec) Name() string {
	return named.data["name"].(string)
}

func (named *namedPullSpec) Image() string {
	return named.data[named.imageKey].(string)
}

func (named *namedPullSpec) Data() map[string]interface{} {
	return named.data
}

func (named *namedPullSpec) SetImage(image string) {
	named.data[named.imageKey] = image
}

func (named *namedPullSpec) AsYamlObject() map[string]interface{} {
	return map[string]interface{}{
		"name":  named.Name(),
		"image": named.Image(),
	}
}

type Container struct {
	namedPullSpec
}

func (container *Container) String() string {
	return fmt.Sprintf("container %s", container.Name())
}

func NewContainer(data interface{}) (*Container, error) {
	dataMap, ok := data.(map[string]interface{})

	if !ok {
		return nil, errors.New("expected map[string]interface{} type")
	}

	return &Container{
		namedPullSpec: namedPullSpec{
			imageKey: "image",
			data:     dataMap,
		},
	}, nil
}

type InitContainer struct {
	namedPullSpec
}

func (container *InitContainer) String() string {
	return fmt.Sprintf("initcontainer %s", container.Name())
}

func NewInitContainer(data interface{}) (*InitContainer, error) {
	dataMap, ok := data.(map[string]interface{})

	if !ok {
		return nil, errors.New("expected map[string]interface{} type")
	}

	return &InitContainer{
		namedPullSpec: namedPullSpec{
			imageKey: "image",
			data:     dataMap,
		},
	}, nil
}

type RelatedImage struct {
	namedPullSpec
}

func (relatedImage *RelatedImage) String() string {
	return fmt.Sprintf("relatedImage %s", relatedImage.Name())
}

func NewRelatedImage(data interface{}) (*RelatedImage, error) {
	dataMap, ok := data.(map[string]interface{})

	if !ok {
		return nil, errors.New("expected map[string]interface{} type")
	}

	return &RelatedImage{
		namedPullSpec: namedPullSpec{
			imageKey: "image",
			data:     dataMap,
		},
	}, nil
}

type RelatedImageEnv struct {
	namedPullSpec
}

func (relatedImage *RelatedImageEnv) String() string {
	return fmt.Sprintf("%s var", relatedImage.Name())
}

func (relatedImage *RelatedImageEnv) Name() string {
	text := fmt.Sprintf("%v", relatedImage.data["name"])
	return strings.ToLower(text[len("RELATED_IMAGE_"):])
}

func NewRelatedImageEnv(data map[string]interface{}) *RelatedImageEnv {
	return &RelatedImageEnv{
		namedPullSpec: namedPullSpec{
			imageKey: "value",
			data:     data,
		},
	}
}

type Annotation struct {
	namedPullSpec
	startI, endI int
}

func NewAnnotation(data map[string]interface{}, key string, startI, endI int) *Annotation {
	return &Annotation{
		namedPullSpec: namedPullSpec{
			imageKey: key,
			data:     data,
		},
		startI: startI,
		endI:   endI,
	}
}

func (annotation *Annotation) Image() string {
	i, j := annotation.startI, annotation.endI
	text := fmt.Sprintf("%v", annotation.data[annotation.imageKey])
	return text[i:j]
}

func (annotation *Annotation) String() string {
	return fmt.Sprintf("annotation %s", annotation.Name())
}

func (annotation *Annotation) SetImage(image string) {
	i, j := annotation.startI, annotation.endI
	text := fmt.Sprintf("%v", annotation.data[annotation.imageKey])
	annotation.data[annotation.imageKey] = fmt.Sprintf("%v%s%v", text[:i], image, text[j:])
}

func (annotation *Annotation) Name() string {
	image := imagename.Parse(annotation.Image())
	tag := image.Tag

	if strings.HasPrefix(tag, "sha256") {
		tag = tag[len("sha256:"):]
	}
	return fmt.Sprintf("%s-%s-annotation", image.Registry, tag)
}

type OperatorCSV struct {
	fs                fs.FS
	path              string
	data              unstructured.Unstructured
	pullspecHeuristic PullSpecHeuristic
}

func NewOperatorCSV(path string, data *unstructured.Unstructured, pullSpecHeuristic PullSpecHeuristic) (*OperatorCSV, error) {
	if data.GetKind() != operatorCsvKind {
		return nil, ErrNotClusterServiceVersion
	}

	if pullSpecHeuristic == nil {
		pullSpecHeuristic = DefaultPullspecHeuristic
	}

	return &OperatorCSV{
		data:              *data,
		path:              path,
		pullspecHeuristic: pullSpecHeuristic,
	}, nil
}

const (
	operatorCsvKind = "ClusterServiceVersion"
)

var (
	ErrNotClusterServiceVersion = errors.New("Not a ClusterServiceVersion")
)

func NewOperatorCSVFromPath(path string, pullSpecHeuristic PullSpecHeuristic) (*OperatorCSV, error) {
	dir, file := filepath.Dir(path), filepath.Base(path)

	if dir == "" {
		dir = "."
	}

	return NewOperatorCSVFromFile(file, os.DirFS(dir), pullSpecHeuristic)
}

func FromDirectory(path string, pullSpecHeuristic PullSpecHeuristic) ([]*OperatorCSV, error) {
	operatorCSVs := []*OperatorCSV{}

	err := filepath.Walk(path, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			log.Printf("prevent panic by handling failure accessing a path %q: %v\n", path, err)
			return err
		}

		log.Println(info.Name(), info.IsDir())

		if info.IsDir() ||
			!(strings.HasSuffix(info.Name(), ".yaml") || strings.HasSuffix(info.Name(), ".yml")) {
			log.Printf("skipping non-yaml file without errors: %+v \n", info.Name())
			return nil
		}

		log.Printf("visited file or dir: %q\n", path)
		csv, err := NewOperatorCSVFromPath(path, pullSpecHeuristic)

		if err != nil && errors.Is(err, ErrNotClusterServiceVersion) {
			log.Printf("skipping file because it's not a ClusterServiceVersion: %+v \n", info.Name())
			return nil
		}

		if err != nil {
			log.Printf("failure reading the file: %+v \n", info.Name())
			return err
		}

		operatorCSVs = append(operatorCSVs, csv)
		return nil
	})

	if err != nil {
		log.Printf("failure walking the directory: %+v \n", err)
		return nil, err
	}

	return operatorCSVs, nil
}

func NewOperatorCSVFromFile(path string, inFs fs.FS, pullSpecHeuristic PullSpecHeuristic) (*OperatorCSV, error) {
	data := &unstructured.Unstructured{}

	fileData, err := fs.ReadFile(inFs, path)

	if err != nil {
		return nil, err
	}

	// decode YAML into unstructured.Unstructured
	dec := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
	_, _, err = dec.Decode(fileData, nil, data)

	if err != nil {
		return nil, err
	}

	csv, err := NewOperatorCSV(path, data, pullSpecHeuristic)

	if err != nil {
		return nil, err
	}

	csv.fs = inFs
	return csv, nil
}

func (csv *OperatorCSV) Dump() error {
	return nil
}

func (csv *OperatorCSV) HasRelatedImages() bool {
	return false
}

func (csv *OperatorCSV) HasRelatedImageEnvs() bool {
	return false
}

func (csv *OperatorCSV) GetPullSpecs() ([]*imagename.ImageName, error) {
	pullspecs := make(map[imagename.ImageName]interface{})

	namedList, err := csv.namedPullSpecs()

	if err != nil {
		return nil, err
	}

	for i := range namedList {
		ps := namedList[i]
		log.Printf("Found pullspec for %s: %s", ps.String(), ps.Image())
		image := imagename.Parse(ps.Image())
		pullspecs[*image] = nil
	}

	imageList := make([]*imagename.ImageName, 0, len(pullspecs))

	for key := range pullspecs {
		localKey := key
		imageList = append(imageList, &localKey)
	}

	return imageList, nil
}

func (csv *OperatorCSV) ReplacePullSpecs(replacement map[imagename.ImageName]imagename.ImageName) error {
	pullspecs, err := csv.namedPullSpecs()
	if err != nil {
		return err
	}

	for _, pullspec := range pullspecs {
		old := imagename.Parse(pullspec.Image())
		new, ok := replacement[*old]

		if ok && *old != new {
			log.Printf("%s - Replaced pullspec for %s: %s -> %s", csv.path, pullspec.String(), *old, new)
			pullspec.SetImage(new.String())
		}
	}

	return nil
}

func (csv *OperatorCSV) ReplacePullSpecsEverywhere(replacement map[imagename.ImageName]imagename.ImageName) error {
	pullspecs := []NamedPullSpec{}
	annotationPullSpecs, err := csv.annotationPullSpecs(knownAnnotationKeys)

	if err != nil {
		return err
	}

	guessedAnnotationPullSpecs, err := csv.annotationPullSpecs(nil)

	if err != nil {
		return err
	}

	pullspecs = append(pullspecs, annotationPullSpecs...)
	pullspecs = append(pullspecs, guessedAnnotationPullSpecs...)

	err = csv.findPotentialPullSpecsNotInAnnotations(csv.data.Object, &pullspecs)

	if err != nil {
		return err
	}

	for _, pullspec := range pullspecs {
		old := imagename.Parse(pullspec.Image())
		new, ok := replacement[*old]

		if ok && *old != new {
			log.Printf("%s - Replaced pullspec for %s: %s -> %s", csv.path, pullspec.String(), *old, new)
			pullspec.SetImage(new.String())
		}
	}

	return nil
}

func (csv *OperatorCSV) SetRelatedImages() {

}

var knownAnnotationKeys = StringSlice{"containerImage"}

func (csv *OperatorCSV) namedPullSpecs() ([]NamedPullSpec, error) {
	pullspecs := []NamedPullSpec{}

	relatedImages, err := csv.relatedImagePullSpecs()

	if err != nil {
		return pullspecs, err
	}

	containers, err := csv.containerPullSpecs()

	if err != nil {
		return pullspecs, err
	}

	initContainers, err := csv.initContainerPullSpecs()

	if err != nil {
		return pullspecs, err
	}

	relatedImageEnvPullSpecs, err := csv.relatedImageEnvPullSpecs()

	if err != nil {
		return pullspecs, err
	}

	annotationPullSpecs, err := csv.annotationPullSpecs(knownAnnotationKeys)

	if err != nil {
		return pullspecs, err
	}

	guessedAnnotationPullSpecs, err := csv.annotationPullSpecs(nil)

	if err != nil {
		return pullspecs, err
	}

	pullspecs = append(pullspecs, relatedImages...)
	pullspecs = append(pullspecs, containers...)
	pullspecs = append(pullspecs, initContainers...)
	pullspecs = append(pullspecs, relatedImageEnvPullSpecs...)
	pullspecs = append(pullspecs, annotationPullSpecs...)
	pullspecs = append(pullspecs, guessedAnnotationPullSpecs...)

	return pullspecs, nil
}

var relatedImagesLens = NewLens().M("spec").M("relatedImages").Build()

func (csv *OperatorCSV) relatedImagePullSpecs() ([]NamedPullSpec, error) {
	lookupResultSlice, err := relatedImagesLens.L(csv.data.Object)

	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return []NamedPullSpec{}, nil
		}

		return nil, err
	}

	pullspecs := make([]NamedPullSpec, 0, len(lookupResultSlice))

	for i := range lookupResultSlice {
		data := lookupResultSlice[i]

		pullspec, err := NewRelatedImage(data)

		if err != nil {
			return nil, err
		}

		pullspecs = append(pullspecs, pullspec)
	}

	return pullspecs, nil
}

func (csv *OperatorCSV) relatedImageEnvPullspecs() ([][]int, error) {
	return nil, nil
}

var deploymentLens = NewLens().M("spec").M("install").M("spec").M("deployments").Build()

func (csv *OperatorCSV) deployments() ([]interface{}, error) {
	return deploymentLens.L(csv.data.Object)
}

var initContainerLens = NewLens().M("spec").M("template").M("spec").M("initContainers").Build()

func (csv *OperatorCSV) initContainerPullSpecs() ([]NamedPullSpec, error) {
	deployments, err := csv.deployments()

	if err != nil {
		return nil, err
	}

	pullspecs := make([]NamedPullSpec, 0, 0)

	for i := range deployments {
		lookupResultSlice, err := initContainerLens.L(deployments[i])
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				continue
			}

			return nil, err
		}

		for i := range lookupResultSlice {
			data := lookupResultSlice[i]

			pullspec, err := NewInitContainer(data)
			if err != nil {
				return nil, err
			}

			pullspecs = append(pullspecs, pullspec)
		}
	}

	return pullspecs, nil
}

var containerLens = NewLens().M("spec").M("template").M("spec").M("containers").Build()

func (csv *OperatorCSV) containerPullSpecs() ([]NamedPullSpec, error) {
	deployments, err := csv.deployments()

	if err != nil {
		return nil, err
	}

	pullspecs := make([]NamedPullSpec, 0, 0)

	for i := range deployments {
		lookupResultSlice, err := containerLens.L(deployments[i])

		if err != nil {
			if errors.Is(err, ErrNotFound) {
				continue
			}

			return nil, err
		}

		for i := range lookupResultSlice {
			data := lookupResultSlice[i]

			pullspec, err := NewContainer(data)

			if err != nil {
				return nil, err
			}

			pullspecs = append(pullspecs, pullspec)
		}
	}

	return pullspecs, nil
}

func (csv *OperatorCSV) relatedImageEnvPullSpecs() ([]NamedPullSpec, error) {
	containers, err := csv.containerPullSpecs()

	if err != nil {
		return nil, err
	}

	initContainers, err := csv.initContainerPullSpecs()

	if err != nil {
		return nil, err
	}

	allContainers := append(containers, initContainers...)

	relatedImageEnvs := []NamedPullSpec{}

	for i := range allContainers {
		c := allContainers[i].Data()

		env, ok := c["env"]

		if !ok {
			continue
		}

		envMaps, ok := env.([]interface{})
		if !ok {
			return nil, errors.New("expected type slice")
		}

		for j := range envMaps {
			envMap, ok := envMaps[j].(map[string]interface{})

			if !ok {
				return nil, errors.New("expected type map")
			}

			// only look at RELATED_IMAGE env vars
			if name, ok := envMap["name"]; !(ok && strings.HasPrefix(name.(string), "RELATED_IMAGE_")) {
				continue
			}

			if _, hasValueFrom := envMap["valueFrom"]; hasValueFrom {
				return nil, NewError(nil, `%s: "valueFrom" references are not supported`, envMap["name"])
			}

			ps := NewRelatedImageEnv(envMap)
			relatedImageEnvs = append(relatedImageEnvs, ps)
		}
	}

	return relatedImageEnvs, nil
}

func (csv *OperatorCSV) annotationPullSpecs(keyFilter StringSlice) ([]NamedPullSpec, error) {
	pullSpecs := []NamedPullSpec{}

	annotationObjects, err := csv.findAllAnnotations()

	if err != nil {
		return nil, err
	}

	for i := range annotationObjects {
		obj := annotationObjects[i]
		for rKey := range obj {
			key := rKey
			val := obj[key]

			if keyFilter != nil && !keyFilter.Contains(key) {
				continue
			}

			valStr := fmt.Sprintf("%v", val)
			results := csv.pullspecHeuristic(valStr)

			for j := range results {
				ii, jj := results[j][0], results[j][1]
				pullSpecs = append(pullSpecs, NewAnnotation(obj, key, ii, jj))
			}
		}
	}

	return NamedPullSpecSlice(pullSpecs).Reverse(), nil
}

var (
	csvAnnotations         = NewLens().M("metadata").M("annotations").Build()
	deploymentAnnotations  = NewLens().M("spec").M("template").M("metadata").M("annotations").Build()
	deploymentsAnnotations = NewLens().
				M("spec").M("install").M("spec").M("deployments").
				Apply(deploymentAnnotations).
				Build()
)

func (csv *OperatorCSV) findAllAnnotations() ([]map[string]interface{}, error) {
	findAnnotationMaps := []func() (map[string]interface{}, error){
		csvAnnotations.MFunc(csv.data.Object),
	}

	findAnnotationSlices := []func() ([]interface{}, error){
		deploymentsAnnotations.LFunc(csv.data.Object),
		func() ([]interface{}, error) {
			results := []interface{}{}
			err := csv.findRandomCSVAnnotations(csv.data.Object, &results, false)
			return results, err
		},
	}

	annotations := []map[string]interface{}{}

	for _, findAnnotation := range findAnnotationMaps {
		result, err := findAnnotation()

		if err != nil {
			if errors.Is(err, ErrNotFound) {
				continue
			}
			return nil, err
		}

		annotations = append(annotations, result)
	}

	for _, findAnnotation := range findAnnotationSlices {
		results, err := findAnnotation()

		if err != nil {
			if errors.Is(err, ErrNotFound) {
				continue
			}
			return nil, err
		}

		for _, result := range results {
			annotationResult := result.(map[string]interface{})
			annotations = append(annotations, annotationResult)
		}
	}

	return annotations, nil
}

var annotations = NewLens().M("metadata").M("annotations").Build()

func (csv *OperatorCSV) findRandomCSVAnnotations(root map[string]interface{}, results *[]interface{}, underMetadata bool) error {
	annos, err := annotations.M(root)

	if err != nil && !errors.Is(err, ErrNotFound) {
		return err
	}

	if err == nil && len(annos) != 0 {
		*results = append(*results, annos)
	}

	for key := range root {
		isUnderMetadata := false

		if key == "metadata" {
			if underMetadata {
				continue
			}

			isUnderMetadata = true
		}

		if slicev, ok := root[key].([]interface{}); ok {

			for i := range slicev {
				if datav, ok := slicev[i].(map[string]interface{}); ok {
					err := csv.findRandomCSVAnnotations(datav, results, isUnderMetadata)

					if err != nil {
						return err
					}
				}
			}
		}

		if datav, ok := root[key].(map[string]interface{}); ok {
			err := csv.findRandomCSVAnnotations(datav, results, isUnderMetadata)

			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (csv *OperatorCSV) findPotentialPullSpecsNotInAnnotations(root map[string]interface{}, specs *[]NamedPullSpec) error {
	for rKey := range root {
		key := rKey
		val := root[key]

		valStr := fmt.Sprintf("%v", val)
		results := csv.pullspecHeuristic(valStr)

		for j := range results {
			ii, jj := results[j][0], results[j][1]
			*specs = append(*specs, NewAnnotation(root, key, ii, jj))
		}
	}

	for key := range root {
		if key == "metadata" {
			continue
		}

		if slicev, ok := root[key].([]interface{}); ok {

			for i := range slicev {
				if datav, ok := slicev[i].(map[string]interface{}); ok {
					err := csv.findPotentialPullSpecsNotInAnnotations(datav, specs)

					if err != nil {
						return err
					}
				}
			}
		}

		if datav, ok := root[key].(map[string]interface{}); ok {
			err := csv.findPotentialPullSpecsNotInAnnotations(datav, specs)

			if err != nil {
				return err
			}
		}
	}

	return nil
}

var (
	ErrNotFound                  = errors.New("path not found")
	ErrPathExpectedDifferentType = errors.New("path expected different type")
)

type errBase struct {
	cause error
	err   error
}

func NewError(cause error, format string, args ...interface{}) error {
	return errBase{
		err:   errors.New(fmt.Sprintf(format, args...)),
		cause: cause,
	}
}

func (e errBase) Error() string {
	return e.err.Error()
}

func (e errBase) Unwrap() error {
	return e.cause
}

type StringSlice []string

func (l StringSlice) Contains(in string) bool {
	for _, key := range l {
		if key == in {
			return true
		}
	}
	return false
}

type NamedPullSpecSlice []NamedPullSpec

func (n NamedPullSpecSlice) Reverse() NamedPullSpecSlice {
	for i := 0; i < len(n)/2; i++ {
		j := len(n) - i - 1
		n[i], n[j] = n[j], n[i]
	}
	return n
}
