package buildrun

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestBuildRun(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "BuildRun Suite")
}
