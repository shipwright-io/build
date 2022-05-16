package tektonrun_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestTektonRun(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "TektonRun Suite")
}
