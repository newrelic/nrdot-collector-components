// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package testbed // import "github.com/newrelic/nrdot-plus-collector-components/testbed/testbed"

import (
	"testing"

	"github.com/shirou/gopsutil/v4/process"
)

type StartParams struct {
	Name         string
	LogFilePath  string
	CmdArgs      []string
	resourceSpec *ResourceSpec
}

type ResourceConsumption struct {
	CPUPercentAvg   float64
	CPUPercentMax   float64
	CPUPercentLimit float64
	RAMMiBAvg       uint32
	RAMMiBMax       uint32
	RAMMiBLimit     uint32
}

// nrdotplustcolRunner defines the interface for configuring, starting and stopping one or more instances of
// nrdotplustcol which will be the subject of testing being executed.
type nrdotplustcolRunner interface {
	// PrepareConfig stores the provided YAML-based nrdotplustcol configuration file in the format needed by the nrdotplustcol
	// instance(s) this runner manages. If successful, it returns the cleanup config function to be executed after
	// the test is executed.
	PrepareConfig(t *testing.T, configStr string) (configCleanup func(), err error)
	// Start starts the nrdotplustcol instance(s) if not already running which is the subject of the test to be run.
	// It returns the host:port of the data receiver to post test data to.
	Start(args StartParams) error
	// Stop stops the nrdotplustcol instance(s) which are the subject of the test just run if applicable. Returns whether
	// the instance was actually stopped or not.
	Stop() (stopped bool, err error)
	// WatchResourceConsumption toggles on the monitoring of resource consumpution by the nrdotplustcol instance under test.
	WatchResourceConsumption() error
	// GetProcessMon returns the Process being used to monitor resource consumption.
	GetProcessMon() *process.Process
	// GetTotalConsumption returns the data collected by the process monitor.
	GetTotalConsumption() *ResourceConsumption
	// GetResourceConsumption returns the data collected by the process monitor as a display string.
	GetResourceConsumption() string
}
