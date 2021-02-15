package buildrun_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestBuildrun(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Buildrun Suite")
}
