package logger

import "go.uber.org/zap"

func New(appEnv string) (*zap.Logger, error) {
	if appEnv == "local" {
		return zap.NewDevelopment()
	}

	return zap.NewProduction()
}
