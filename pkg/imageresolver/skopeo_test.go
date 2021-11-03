package imageresolver

import (
	"errors"
	"log"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/stretchr/testify/mock"
)

var _ = Describe("skopeo image resolver", func() {
	var sut *Skopeo
	var mockRunner *mockCommandRunner
	var mockProvider *mockCommandRunnerProvider

	BeforeEach(func() {
		log.SetOutput(GinkgoWriter)
		mockRunner = new(mockCommandRunner)
		mockProvider = new(mockCommandRunnerProvider)

		mockProvider.On("Command", "skopeo", mock.Anything).Return(mockRunner)

		sut = &Skopeo{
			path:     "skopeo",
			authFile: "nonexistantfile",
			command:  mockProvider.Command,
		}
	})

	const imageName = "example.com/foo/bar@sha256:c5d902c53b4afcf32ad746fd9d696431650d3fbe8f7b10ca10519543fefd772c"

	It("should use raw if version 2", func() {
		mockRunner.On("CombinedOutput").Return([]byte(`{"schemaVersion": 2}`), nil)

		expected := imageName
		resolved, err := sut.ResolveImageReference("example.com/foo/bar:latest")
		Expect(err).To(Succeed())
		Expect(resolved).To(Equal(expected))
		mockProvider.AssertExpectations(GinkgoT())
		Expect(mockProvider.Calls[0].Arguments.Get(1)).To(ContainElement("--raw"))
		Expect(mockProvider.Calls[0].Arguments.Get(1)).To(ContainElements("--authfile", "nonexistantfile"))
	})

	It("should make 2 calls if version 1", func() {
		mockRunner.On("CombinedOutput").Return([]byte(`{"schemaVersion": 1}`), nil).Once()
		mockRunner.On("CombinedOutput").Return([]byte(`{"Digest": "sha256:1"}`), nil).Once()

		expected := "example.com/foo/bar@sha256:1"
		resolved, err := sut.ResolveImageReference("example.com/foo/bar:latest")
		Expect(err).To(Succeed())
		Expect(resolved).To(Equal(expected))
		mockProvider.AssertExpectations(GinkgoT())
		Expect(mockProvider.Calls[0].Arguments.Get(1)).To(ContainElement("--raw"))
		Expect(mockProvider.Calls[1].Arguments.Get(1)).To(Not(ContainElement("--raw")))
	})

	It("should not change if digest", func() {
		mockProvider.On("Command", "skopeo", mock.Anything).Return(mockRunner)
		mockRunner.On("Run").Return(nil)
		mockRunner.On("CombinedOutput").Return([]byte(`{"schemaVersion": 2}`), nil)

		reference := imageName
		resolved, err := sut.ResolveImageReference(reference)
		Expect(err).To(Succeed())
		Expect(resolved).To(Equal(reference))
		mockProvider.AssertExpectations(GinkgoT())
		Expect(mockProvider.Calls[0].Arguments.Get(1)).To(ContainElement("--raw"))
	})

	It("should use an authfile", func() {
		mockRunner.On("CombinedOutput").Return([]byte(`{"schemaVersion": 2}`), nil).Once()

		tmpDir := os.TempDir()
		f, err := os.CreateTemp(tmpDir, "authfile")
		Expect(err).To(Succeed())
		defer f.Close()

		_, err = f.WriteString("spam")
		Expect(err).To(Succeed())
		sut.authFile = filepath.Join(tmpDir, f.Name())

		expected := imageName
		resolved, err := sut.ResolveImageReference("example.com/foo/bar:latest")
		Expect(resolved).To(Equal(expected))
		mockProvider.AssertExpectations(GinkgoT())
		Expect(mockProvider.Calls[0].Arguments.Get(1)).To(ContainElements("--authfile", sut.authFile))
	})

	It("should fail if authfile doesn't exist", func() {
		tmpDir := os.TempDir()
		fileName := filepath.Join(tmpDir, "fakeFile")

		_, err := NewSkopeoResolver("path", fileName)
		Expect(err).To(HaveOccurred())
	})

	It("should retry", func() {
		mockRunner.On("CombinedOutput").Return([]byte{}, errors.New("failed")).Once()
		mockRunner.On("CombinedOutput").Return([]byte{}, errors.New("failed")).Once()
		mockRunner.On("CombinedOutput").Return([]byte(`{"schemaVersion": 2}`), nil).Once()

		expected := imageName
		resolved, err := sut.ResolveImageReference("example.com/foo/bar:latest")
		Expect(err).To(Succeed())
		Expect(resolved).To(Equal(expected))
		mockProvider.AssertExpectations(GinkgoT())
		Expect(mockProvider.Calls[0].Arguments.Get(1)).To(ContainElement("--raw"))
		Expect(mockProvider.Calls[1].Arguments.Get(1)).To(ContainElement("--raw"))
		Expect(mockProvider.Calls[2].Arguments.Get(1)).To(ContainElement("--raw"))
	})

	It("should retry and fail", func() {
		mockRunner.On("CombinedOutput").Return([]byte{}, errors.New("failed")).Once()
		mockRunner.On("CombinedOutput").Return([]byte{}, errors.New("failed")).Once()
		mockRunner.On("CombinedOutput").Return([]byte{}, errors.New("failed")).Once()

		_, err := sut.ResolveImageReference("example.com/foo/bar:latest")
		Expect(err).To(HaveOccurred())
		mockProvider.AssertExpectations(GinkgoT())
		Expect(mockProvider.Calls).To(HaveLen(3))
		Expect(mockProvider.Calls[0].Arguments.Get(1), ContainElement("--raw"))
		Expect(mockProvider.Calls[1].Arguments.Get(1), ContainElement("--raw"))
		Expect(mockProvider.Calls[2].Arguments.Get(1), ContainElement("--raw"))
	})

})

type mockCommandRunnerProvider struct {
	mock.Mock
}

func (m *mockCommandRunnerProvider) Command(name string, arg ...string) commandRunner {
	args := m.Called(name, arg)
	return args.Get(0).(commandRunner)
}

type mockCommandRunner struct {
	mock.Mock
}

func (m *mockCommandRunner) Run() error {
	args := m.Called()
	return args.Error(0)
}

func (m *mockCommandRunner) CombinedOutput() ([]byte, error) {
	args := m.Called()
	return args.Get(0).([]byte), args.Error(1)
}
