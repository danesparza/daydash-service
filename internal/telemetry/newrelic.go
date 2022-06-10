package telemetry

import (
	"os"

	"github.com/danesparza/daydash-service/internal/logger"
	"github.com/newrelic/go-agent/v3/newrelic"
	"go.uber.org/zap"
)

var (
	rootlogger *zap.Logger
	zlog       *zap.SugaredLogger
	NRLicense  = "Unknown"
	NRAppName  = "api.daydash.net"
	NRApp      = &newrelic.Application{}
)

func init() {

	rootlogger, _ = logger.NewProd()
	defer rootlogger.Sync() // flushes buffer, if any
	zlog = rootlogger.Sugar()

	err := *new(error)

	//	If we have NR environment variables, use them:
	if os.Getenv("NR_DASHBOARD_LIC") != "" {
		NRLicense = os.Getenv("NR_DASHBOARD_LIC")
	}

	if os.Getenv("NR_DASHBOARD_APP") != "" {
		NRAppName = os.Getenv("NR_DASHBOARD_APP")
	}

	NRApp, err = newrelic.NewApplication(
		newrelic.ConfigAppName(NRAppName),
		newrelic.ConfigLicense(NRLicense),
		newrelic.ConfigDistributedTracerEnabled(true),
	)

	if err != nil {
		zlog.Errorw(
			"Error trying to create New Relic connection",
			"App name", NRApp,
			"License information", NRLicense,
			"error", err,
		)
	}
}
