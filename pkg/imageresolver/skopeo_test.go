package imageresolver

import (
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/stretchr/testify/mock"
)

var _ = Describe("skopeo image resolver", func() {
	var sut *SkopeoImageResolver
	var mockRunner *mockCommandRunner
	var mockProvider *mockCommandRunnerProvider

	BeforeEach(func() {
		mockRunner = new(mockCommandRunner)
		mockProvider = new(mockCommandRunnerProvider)

		mockProvider.On("Command", "skopeo", mock.Anything).Return(mockRunner)

		sut = &SkopeoImageResolver{
			path:     "skopeo",
			authFile: "nonexistantfile",
			command:  mockProvider.Command,
		}
	})

	It("should use raw if version 2", func() {
		mockRunner.On("Run").Return(nil)
		mockRunner.On("Output").Return([]byte(`{"schemaVersion": 2}`), nil)

		expected := "example.com/foo/bar@sha256:c5d902c53b4afcf32ad746fd9d696431650d3fbe8f7b10ca10519543fefd772c"
		resolved, err := sut.ResolveImageReference("example.com/foo/bar:latest")
		Expect(err).To(Succeed())
		Expect(resolved).To(Equal(expected))
		mockProvider.AssertExpectations(GinkgoT())
		Expect(mockProvider.Calls[0].Arguments.Get(1), ConsistOf("--raw"))
	})

	It("should make 2 calls if version 1", func() {
		mockRunner.On("Run").Return(nil)
		mockRunner.On("Output").Return([]byte(`{"schemaVersion": 1}`), nil).Once()
		mockRunner.On("Output").Return([]byte(`{"Digest": "sha256:1"}`), nil).Once()

		expected := "example.com/foo/bar@sha256:1"
		resolved, err := sut.ResolveImageReference("example.com/foo/bar:latest")
		Expect(err).To(Succeed())
		Expect(resolved).To(Equal(expected))
		mockProvider.AssertExpectations(GinkgoT())
		Expect(mockProvider.Calls[0].Arguments.Get(1), ConsistOf("--raw"))
		Expect(mockProvider.Calls[1].Arguments.Get(1)).To(Not(ConsistOf("--raw")))
	})

	It("should not change if digest", func() {
		mockProvider.On("Command", "skopeo", mock.Anything).Return(mockRunner)
		mockRunner.On("Run").Return(nil)
		mockRunner.On("Output").Return([]byte(`{"schemaVersion": 2}`), nil)

		reference := "example.com/foo/bar@sha256:c5d902c53b4afcf32ad746fd9d696431650d3fbe8f7b10ca10519543fefd772c"
		resolved, err := sut.ResolveImageReference(reference)
		Expect(err).To(Succeed())
		Expect(resolved).To(Equal(reference))
		mockProvider.AssertExpectations(GinkgoT())
		Expect(mockProvider.Calls[0].Arguments.Get(1), ConsistOf("--raw"))
	})

	It("should retry", func() {
		mockRunner.On("Run").Return(nil)
		mockRunner.On("Output").Return([]byte{}, errors.New("failed")).Once()
		mockRunner.On("Output").Return([]byte{}, errors.New("failed")).Once()
		mockRunner.On("Output").Return([]byte(`{"schemaVersion": 2}`), nil).Once()

		expected := "example.com/foo/bar@sha256:c5d902c53b4afcf32ad746fd9d696431650d3fbe8f7b10ca10519543fefd772c"
		resolved, err := sut.ResolveImageReference("example.com/foo/bar:latest")
		Expect(err).To(Succeed())
		Expect(resolved).To(Equal(expected))
		mockProvider.AssertExpectations(GinkgoT())
		Expect(mockProvider.Calls[0].Arguments.Get(1), ConsistOf("--raw"))
		Expect(mockProvider.Calls[1].Arguments.Get(1), ConsistOf("--raw"))
		Expect(mockProvider.Calls[2].Arguments.Get(1), ConsistOf("--raw"))
	})

	It("should retry and fail", func() {
		mockRunner.On("Run").Return(nil)
		mockRunner.On("Output").Return([]byte{}, errors.New("failed")).Once()
		mockRunner.On("Output").Return([]byte{}, errors.New("failed")).Once()
		mockRunner.On("Output").Return([]byte{}, errors.New("failed")).Once()

		_, err := sut.ResolveImageReference("example.com/foo/bar:latest")
		Expect(err).To(HaveOccurred())
		mockProvider.AssertExpectations(GinkgoT())
		Expect(mockProvider.Calls[0].Arguments.Get(1), ConsistOf("--raw"))
		Expect(mockProvider.Calls[1].Arguments.Get(1), ConsistOf("--raw"))
		Expect(mockProvider.Calls[2].Arguments.Get(1), ConsistOf("--raw"))
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

func (m *mockCommandRunner) Output() ([]byte, error) {
	args := m.Called()
	return args.Get(0).([]byte), args.Error(1)
}
