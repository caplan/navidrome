package taglib

import (
	"testing"

	"github.com/caplan/navidrome/log"
	"github.com/caplan/navidrome/tests"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestTagLib(t *testing.T) {
	tests.Init(t, true)
	log.SetLevel(log.LevelFatal)
	RegisterFailHandler(Fail)
	RunSpecs(t, "TagLib Suite")
}
