package main

import (
	"fmt"
	"math"
	"os"
	"testing"
	"time"

	. "github.com/slukits/gounit"
)

type AnAdapter struct{ Suite }

func (s *AnAdapter) SetUp(t *T) { t.Parallel() }

func (s *AnAdapter) Is_not_activated_if_state_retrieval_fails(t *T) {
	dev, err := mockAdapterIsNotActive(t, &Env{}).Device()
	t.FatalOn(err)
	t.Not.True(dev.IsActivated())
}

func (s *AnAdapter) Scan_fails_on_system_bus_connection_failure(t *T) {
	adapter := mckAdapterBusFailure(t)
	_, err := adapter.Scan()
	t.ErrIs(err, ErrAdapterScan)
	t.ErrIs(err, ErrMckAdapterBusConnectionFailure)
}

func (s *AnAdapter) Scan_fails_on_connection_close_failure(t *T) {
	adapter := mckAdapterCnnCloseFailure(t)
	_, err := adapter.Scan()
	t.ErrIs(err, ErrAdapterScan)
	t.ErrIs(err, ErrMckAdapterCnnCloseFailure)
}

func (s *AnAdapter) Scan_fails_on_signal_matcher_setup_failure(t *T) {
	adapter := mckAdapterSignalMatcherFailure(t)
	_, err := adapter.Scan()
	t.ErrIs(err, ErrAdapterScan)
	t.ErrIs(err, ErrMckAdapterSignalMatcherFailure)
}

func (s *AnAdapter) Scan_fails_on_device_scan_failure(t *T) {
	adapter := mckAdapterScanFails(t)
	_, err := adapter.Scan()
	t.ErrIs(err, ErrAdapterScan)
	t.ErrIs(err, ErrMckAdapterScanFails)
}

func (s *AnAdapter) Scan_fails_on_last_scan_change_timeout(t *T) {
	adapter, err := (&Env{}).Device()
	t.FatalOn(err)
	adapter.Timeout = 0 * time.Second
	_, err = adapter.Scan()
	t.ErrIs(err, ErrAdapterScan)
	t.ErrIs(err, ErrAdapterPropertyChangeTimeout)
}

func (s *AnAdapter) Scan_fails_on_access_points_retrieval_failure(t *T) {
	adapter := mckAdapterAccessPointsFailure(t)
	_, err := adapter.Scan()
	t.ErrIs(err, ErrAdapterScan)
	t.ErrIs(err, ErrMckAdapterAccessPointsFailure)
}

func (s *AnAdapter) Scan_fails_on_SSID_retrieval_failure(t *T) {
	adapter := mckAdapterSSIDFailure(t)
	_, err := adapter.Scan()
	t.ErrIs(err, ErrAdapterScan)
	t.ErrIs(err, ErrMckAdapterSSIDFailure)
}

func (s *AnAdapter) Scan_fails_on_signal_strength_retrieval_failure(
	t *T,
) {
	adapter := mckAdapterSignalStrengthFailure(t)
	_, err := adapter.Scan()
	t.ErrIs(err, ErrAdapterScan)
	t.ErrIs(err, ErrMckAdapterSignalStrengthFailure)
}

func (s *AnAdapter) Scan_provides_access_points_descending_by_strength(
	t *T,
) {
	_, adapter := mockedDevice(t)
	aa, err := adapter.Scan()
	t.FatalOn(err)
	last := uint8(math.MaxInt8)
	for _, a := range aa {
		t.FatalIfNot(t.True(a.Strength <= last))
		last = a.Strength
	}
	t.True(len(aa) > 0)
}

func TestAnAdapter(t *testing.T) {
	t.Parallel()
	Run(&AnAdapter{}, t)
}

func TestDisconnectAndReconnect(t_ *testing.T) {
	t := NewT(t_)
	if os.Args[len(os.Args)-1] != "all" {
		fmt.Println("TestDisconnectAndReconnect skipped; " +
			"call 'go test -args all' to include this test")
		t_.SkipNow()
	}
	_, adapter := mockedDevice(t)
	ssid, err := adapter.Active()
	t.FatalOn(err)
	t.FatalOn(adapter.Disconnect())
	t.Not.True(adapter.IsActivated())
	t.FatalOn(adapter.Connect(ssid))
	t.True(adapter.IsActivated())
}
