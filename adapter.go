package main

import (
	"errors"
	"fmt"
	"sort"
	"time"

	nm "github.com/Wifx/gonetworkmanager/v2"
	"github.com/godbus/dbus/v5"
)

// BusConnection reduces the dbus.Conn-API to the needs of WifiAdapter
// and makes these features mockable
type BusConnection interface {
	AddMatchSignal(...dbus.MatchOption) error
	Signal(chan<- *dbus.Signal)
	RemoveSignal(chan<- *dbus.Signal)
	Close() error
}

type WifiAdapter struct {
	Lib     AdapterLib
	Timeout time.Duration
	name    string
	dev     nm.DeviceWireless
	libInit bool
}

func (a *WifiAdapter) lib() AdapterLib {
	if !a.libInit {
		a.libInit = true
		if a.Lib.SystemBus == nil {
			a.Lib.SystemBus = func() (BusConnection, error) {
				return dbus.ConnectSystemBus()
			}
		}
		if a.Lib.WaitForPropertyChange == nil {
			a.Lib.WaitForPropertyChange = a.waitForPropertyChange
		}
		if a.Lib.Disconnect == nil {
			a.Lib.Disconnect = a.dev.Disconnect
		}
	}
	return a.Lib
}

func (a *WifiAdapter) IsActivated() bool {
	state, err := a.dev.GetPropertyState()
	if err != nil {
		return false
	}
	return state == nm.NmDeviceStateActivated
}

var ErrAdapterScan = errors.New("wifi-adapter: scan error")

type AccessPoint struct {
	SSID     string
	Strength uint8
}

func (a *WifiAdapter) Scan() (_ []AccessPoint, err error) {
	cnn, err := a.lib().SystemBus()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrAdapterScan, err)
	}
	defer func() {
		if e := cnn.Close(); e != nil {
			err = fmt.Errorf("%w: %w", ErrAdapterScan, e)
		}
	}()
	c := make(chan *dbus.Signal, 100)
	defer close(c)
	if err := a.setupScanSignalMatcher(c, cnn); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrAdapterScan, err)
	}
	defer cnn.RemoveSignal(c)
	if err := a.dev.RequestScan(); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrAdapterScan, err)
	}
	if err := a.lib().WaitForPropertyChange(c, "LastScan"); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrAdapterScan, err)
	}
	aa, err := a.dev.GetPropertyAccessPoints()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrAdapterScan, err)
	}
	accessPoints := []AccessPoint{}
	for _, ap := range aa {
		ssid, err := ap.GetPropertySSID()
		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrAdapterScan, err)
		}
		strength, err := ap.GetPropertyStrength()
		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrAdapterScan, err)
		}
		accessPoints = append(accessPoints, AccessPoint{
			SSID: ssid, Strength: strength})
	}
	sort.Slice(accessPoints, func(i, j int) bool {
		return accessPoints[i].SSID < accessPoints[j].SSID
	})
	sort.SliceStable(accessPoints, func(i, j int) bool {
		return accessPoints[i].Strength > accessPoints[j].Strength
	})
	return accessPoints, nil
}

func (a *WifiAdapter) Disconnect() error {
	return a.lib().Disconnect()
}

var ErrAdapterLastScanChangeTimeout = errors.New(
	"last scan property change timeout")

func (a *WifiAdapter) waitForPropertyChange(
	c chan *dbus.Signal, property string,
) error {
	for {
		select {
		case s := <-c:
			if len(s.Body) < 2 {
				continue
			}
			bodyMap, ok := s.Body[1].(map[string]dbus.Variant)
			if !ok {
				continue
			}
			_, ok = bodyMap[property]
			if !ok {
				continue
			}
			return nil
		case <-time.After(a.Timeout):
			return ErrAdapterLastScanChangeTimeout
		}
	}
}

const DBusProperties = "org.freedesktop.DBus.Properties"
const PropertiesChanged = "PropertiesChanged"

func (a *WifiAdapter) setupScanSignalMatcher(
	c chan *dbus.Signal, cnn BusConnection,
) error {
	err := cnn.AddMatchSignal(
		dbus.WithMatchObjectPath(a.dev.GetPath()),
		dbus.WithMatchInterface(DBusProperties),
		dbus.WithMatchMember(PropertiesChanged),
	)
	if err != nil {
		return err
	}
	cnn.Signal(c)
	return nil
}

func (a *WifiAdapter) DeviceName() string { return a.name }

type AdapterLib struct {
	SystemBus             func() (BusConnection, error)
	WaitForPropertyChange func(chan *dbus.Signal, string) error
	Disconnect            func() error
}
