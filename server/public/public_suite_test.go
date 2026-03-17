package public

import (
	"testing"

	"github.com/caplan/navidrome/log"
	"github.com/caplan/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestPublicEndpoints(t *testing.T) {
	tests.Init(t, false)
	log.SetLevel(log.LevelFatal)
	RegisterFailHandler(Fail)
	RunSpecs(t, "Public Endpoints Suite")
}
