package configtest

import "github.com/caplan/navidrome/conf"

func SetupConfig() func() {
	oldValues := *conf.Server
	return func() {
		conf.Server = &oldValues
	}
}
