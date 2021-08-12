package cmd

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/spf13/cobra"
)

var _ = Describe("Vars", func() {
	Describe("InputParam", func() {
		var (
			sut     InputParam
			testCmd *cobra.Command
		)

		BeforeEach(func() {
			sut = NewInputParam()
			testCmd = &cobra.Command{}
		})

		It("should add to a command", func() {
			Expect(testCmd.Flags().HasFlags()).To(BeFalse())
			sut.AddFlag(testCmd, "test", "-", "test desc")
			Expect(testCmd.Flags().HasFlags()).To(BeTrue())
		})

		It("should and open a file if a valid filepath", func() {
			f, err := ioutil.TempFile(os.TempDir(), "file-*.txt")
			Expect(err).To(Succeed())
			defer os.Remove(f.Name())

			ioutil.WriteFile(f.Name(), []byte("foo"), 0666)

			sut.Name = f.Name()

			err = sut.Init(testCmd, []string{})
			Expect(err).To(Succeed())

			b, err := io.ReadAll(&sut)
			Expect(err).To(Succeed())
			Expect(string(b)).To(Equal("foo"))

			err = sut.Close()
			Expect(err).To(Succeed())
		})

		It("should use the cmd stdin if named -", func() {
			sut.Name = "-"
			buff := bytes.Buffer{}
			
			io.WriteString(&buff, "foo")
			testCmd.SetIn(&buff)
			
			err := sut.Init(testCmd, []string{})
			Expect(err).To(Succeed())
			
			b, err := io.ReadAll(&sut)
			Expect(err).To(Succeed())
			Expect(string(b)).To(Equal("foo"))
			
			err = sut.Close()
			Expect(err).To(Succeed())
		})
	})

	Describe("OutputParam", func() {
		var (
			sut     OutputParam
			testCmd *cobra.Command
		)

		BeforeEach(func() {
			sut = NewOutputParam()
			testCmd = &cobra.Command{}
		})

		It("should add to a command", func() {
			Expect(testCmd.Flags().HasFlags()).To(BeFalse())
			sut.AddFlag(testCmd, "test", "-", "test desc")
			Expect(testCmd.Flags().HasFlags()).To(BeTrue())
		})

		It("should and open a file if a valid filepath", func() {
			tempDir := os.TempDir()
			sut.Name = filepath.Join(tempDir, "test-file.txt")

			err := sut.Init(testCmd, []string{})
			Expect(err).To(Succeed())

			_, err = io.WriteString(&sut, "foo")
			Expect(err).To(Succeed())
		
			err = sut.Close()
			Expect(err).To(Succeed())

			b, err := os.ReadFile(sut.Name)
			Expect(err).To(Succeed())
			
			Expect(string(b)).To(Equal("foo"))
		})

		It("should use the cmd stdout if named -", func() {
			sut.Name = "-"
			buff := bytes.Buffer{}
			
			testCmd.SetOut(&buff)
			
			err := sut.Init(testCmd, []string{})
			Expect(err).To(Succeed())
			
			io.WriteString(&sut, "foo")
			
			Expect(buff.String()).To(Equal("foo"))

			err = sut.Close()
			Expect(err).To(Succeed())
		})
	})
})
