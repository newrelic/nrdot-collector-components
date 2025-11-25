// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package testbed // import "github.com/newrelic/nrdot-plus-collector-components/testbed/testbed"

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/shirou/gopsutil/v4/process"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/provider/fileprovider"
	"go.opentelemetry.io/collector/nrdotplustcol"

	"github.com/newrelic/nrdot-plus-collector-components/internal/common/testutil"
)

// inProcessCollector implements the nrdotplustcolRunner interfaces running a single nrdotplustcol as a go routine within the
// same process as the test executor.
type inProcessCollector struct {
	factories  nrdotplustcol.Factories
	configStr  string
	svc        *nrdotplustcol.Collector
	stopped    bool
	configFile string
	wg         sync.WaitGroup
	t          *testing.T
}

// NewInProcessCollector creates a new inProcessCollector using the supplied component factories.
func NewInProcessCollector(factories nrdotplustcol.Factories) nrdotplustcolRunner {
	return &inProcessCollector{
		factories: factories,
	}
}

func (ipp *inProcessCollector) PrepareConfig(t *testing.T, configStr string) (configCleanup func(), err error) {
	configCleanup = func() {
		// NoOp
	}
	ipp.configStr = configStr
	ipp.t = t
	return configCleanup, err
}

func (ipp *inProcessCollector) Start(StartParams) error {
	var err error
	confFile, err := os.CreateTemp(testutil.TempDir(ipp.t), "conf-")
	if err != nil {
		return err
	}

	if _, err = confFile.WriteString(ipp.configStr); err != nil {
		os.Remove(confFile.Name())
		return err
	}
	ipp.configFile = confFile.Name()

	settings := nrdotplustcol.CollectorSettings{
		BuildInfo: component.NewDefaultBuildInfo(),
		Factories: func() (nrdotplustcol.Factories, error) { return ipp.factories, nil },
		ConfigProviderSettings: nrdotplustcol.ConfigProviderSettings{
			ResolverSettings: confmap.ResolverSettings{
				URIs:              []string{ipp.configFile},
				ProviderFactories: []confmap.ProviderFactory{fileprovider.NewFactory()},
			},
		},
		SkipSettingGRPCLogger: true,
	}

	ipp.svc, err = nrdotplustcol.NewCollector(settings)
	if err != nil {
		return err
	}

	ipp.wg.Add(1)
	go func() {
		defer ipp.wg.Done()
		if appErr := ipp.svc.Run(context.Background()); appErr != nil {
			// TODO: Pass this to the error handler.
			panic(appErr)
		}
	}()

	for {
		switch state := ipp.svc.GetState(); state {
		case nrdotplustcol.StateStarting:
			time.Sleep(time.Second)
		case nrdotplustcol.StateRunning:
			return nil
		default:
			return fmt.Errorf("unable to start, nrdotplustcol state is %d", state)
		}
	}
}

func (ipp *inProcessCollector) Stop() (stopped bool, err error) {
	if !ipp.stopped {
		ipp.stopped = true
		ipp.svc.Shutdown()
		// Do not delete temporary files on Windows because it fails too much on scoped tests.
		// See https://github.com/newrelic/nrdot-plus-collector-components/issues/42639
		if runtime.GOOS != "windows" {
			require.NoError(ipp.t, os.Remove(ipp.configFile))
		}
	}
	ipp.wg.Wait()
	stopped = ipp.stopped
	return stopped, err
}

func (*inProcessCollector) WatchResourceConsumption() error {
	return nil
}

func (*inProcessCollector) GetProcessMon() *process.Process {
	return nil
}

func (*inProcessCollector) GetTotalConsumption() *ResourceConsumption {
	return &ResourceConsumption{
		CPUPercentAvg:   0,
		CPUPercentMax:   0,
		CPUPercentLimit: 0,
		RAMMiBAvg:       0,
		RAMMiBMax:       0,
		RAMMiBLimit:     0,
	}
}

func (*inProcessCollector) GetResourceConsumption() string {
	return ""
}
