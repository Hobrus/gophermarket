package e2e_test

import (
	"os"
	"testing"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
)

func TestE2E(t *testing.T) {
	if os.Getenv("ENABLE_E2E") == "" {
		t.Skip("skipping E2E tests; set ENABLE_E2E=1 to enable")
	}
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "E2E Suite")
}
