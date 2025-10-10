package app

import "go.uber.org/fx"

func Shutdown(shutdowner fx.Shutdowner) {
	shutdowner.Shutdown()
}
