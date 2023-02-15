package main

import (
	"errors"
	"sync"
	"time"

	nm "github.com/Wifx/gonetworkmanager/v2"
	"github.com/godbus/dbus/v5"
	"github.com/slukits/gounit"
)

type MckAdapterIsNotActive struct {
	nm.DeviceWireless
	secondCall bool
}

func mockAdapterIsNotActive(t *gounit.T, env *Env) *Env {
	env.Lib.NewNM = func() (nm.NetworkManager, error) {
		nm_ := mockedNM(t)
		nm_.(*NMMock).allDevices = failIAllDevicesIfNoWifi(
			t, nm_.(*NMMock).NetworkManager)
		return nm_, nil
	}
	env.Lib.NewWifiDevice =
		func(op dbus.ObjectPath) (nm.DeviceWireless, error) {
			wd, err := nm.NewDeviceWireless(op)
			if err != nil {
				t.Fatalf("mock: nm: wifi-device: state err: %v", err)
			}
			return &MckAdapterIsNotActive{DeviceWireless: wd}, nil
		}
	return env
}

func (m *MckAdapterIsNotActive) GetPropertyState() (
	nm.NmDeviceState, error,
) {
	if !m.secondCall {
		m.secondCall = true
		return m.DeviceWireless.GetPropertyState()
	}
	return 0, errors.New("mocked state error")
}

func mckAdapterEnv(t *gounit.T) *Env {
	// fail test if no wifi adapter available
	env := &Env{}
	env.Lib.NewNM = func() (nm.NetworkManager, error) {
		nm_ := mockedNM(t)
		nm_.(*NMMock).allDevices = failIAllDevicesIfNoWifi(
			t, nm_.(*NMMock).NetworkManager)
		return nm_, nil
	}
	return env
}

var ErrMckAdapterBusConnectionFailure = errors.New(
	"bus connection error mock")

func mckAdapterBusFailure(t *gounit.T) *WifiAdapter {
	adapter, err := mckAdapterEnv(t).Device()
	t.FatalOn(err)
	adapter.Lib.SystemBus = func() (BusConnection, error) {
		return nil, ErrMckAdapterBusConnectionFailure
	}
	return adapter
}

/*
Understandably the NetworkManager is not not very happy being flooded
with scan requests.  Hence we need to mock-up the DeviceWireless method
RequestScan and the the library function WaitForPropertyChange to be
run only once per test run.
*/

var scanner = &sync.Mutex{}
var hasScanned = false
var hasWaited = false
var wifiConstructorMocked = false

type MckDevice struct {
	nm.DeviceWireless
	requestScan             func() error
	getPropertyAccessPoints func() ([]nm.AccessPoint, error)
}

var mckAccessPoints []nm.AccessPoint

func mckDevices(t *gounit.T) *Env {
	env := mckAdapterEnv(t)
	env.Lib.NewWifiDevice =
		func(op dbus.ObjectPath) (nm.DeviceWireless, error) {
			wd, err := nm.NewDeviceWireless(op)
			if err != nil {
				t.Fatalf("mock: wifi-device: %v", err)
			}
			return &MckDevice{DeviceWireless: wd}, nil
		}
	env.Lib.NewWifiAdapter =
		func(d nm.DeviceWireless, n string) *WifiAdapter {
			adapter := &WifiAdapter{
				Timeout: 5 * time.Second,
				name:    n,
				dev:     d,
			}
			lib := adapter.lib()
			defaultWaitForPropertyChange := lib.WaitForPropertyChange
			adapter.Lib.WaitForPropertyChange =
				func(c chan *dbus.Signal, s string) error {
					scanner.Lock()
					defer scanner.Unlock()
					if hasWaited {
						return nil
					}
					hasWaited = true
					err := defaultWaitForPropertyChange(c, s)
					t.FatalOn(err)
					aa, err := adapter.dev.(*MckDevice).DeviceWireless.
						GetPropertyAccessPoints()
					t.FatalOn(err)
					mckAccessPoints = aa
					return nil
				}
			return adapter
		}
	return env
}

func (m *MckDevice) RequestScan() error {
	if m.requestScan != nil {
		return m.requestScan()
	}
	scanner.Lock()
	defer scanner.Unlock()
	if hasScanned {
		return nil
	}
	hasScanned = true
	return m.DeviceWireless.RequestScan()
}

func (m *MckDevice) GetPropertyAccessPoints() ([]nm.AccessPoint, error) {
	if m.getPropertyAccessPoints != nil {
		return m.getPropertyAccessPoints()
	}
	scanner.Lock()
	defer scanner.Unlock()
	return mckAccessPoints, nil
}

func mockedDevice(t *gounit.T) (*Env, *WifiAdapter) {
	env := mckDevices(t)
	adapter, err := env.Device()
	t.FatalOn(err)
	return env, adapter
}

type MckAdapterCnnCloseErr struct{ *dbus.Conn }

func mckAdapterCnnCloseFailure(t *gounit.T) *WifiAdapter {
	_, adapter := mockedDevice(t)
	adapter.Lib.SystemBus = func() (BusConnection, error) {
		cnn, err := dbus.ConnectSystemBus()
		t.FatalOn(err)
		return &MckAdapterCnnCloseErr{Conn: cnn}, nil
	}
	return adapter
}

var ErrMckAdapterCnnCloseFailure = errors.New(
	"bus connection close error mock")

func (m *MckAdapterCnnCloseErr) Close() error { return ErrMckAdapterCnnCloseFailure }

type MckAdapterSignalMatcherFailure struct{ *dbus.Conn }

func mckAdapterSignalMatcherFailure(t *gounit.T) *WifiAdapter {
	adapter, err := mckAdapterEnv(t).Device()
	t.FatalOn(err)
	adapter.Lib.SystemBus = func() (BusConnection, error) {
		cnn, err := dbus.ConnectSystemBus()
		t.FatalOn(err)
		return &MckAdapterSignalMatcherFailure{Conn: cnn}, nil
	}
	return adapter
}

var ErrMckAdapterSignalMatcherFailure = errors.New(
	"adapter signal matcher failure mock")

func (m *MckAdapterSignalMatcherFailure) AddMatchSignal(
	...dbus.MatchOption,
) error {
	return ErrMckAdapterSignalMatcherFailure
}

var ErrMckAdapterScanFails = errors.New("adapter scan fails mock")

func mckAdapterScanFails(t *gounit.T) *WifiAdapter {
	_, adapter := mockedDevice(t)
	adapter.dev.(*MckDevice).requestScan = func() error {
		return ErrMckAdapterScanFails
	}
	return adapter
}

type MckAdapterAccessPointsFailure struct{ nm.DeviceWireless }

var ErrMckAdapterAccessPointsFailure = errors.New(
	"adapter scan fails mock")

func mckAdapterAccessPointsFailure(t *gounit.T) *WifiAdapter {
	_, adapter := mockedDevice(t)
	adapter.dev.(*MckDevice).getPropertyAccessPoints =
		func() ([]nm.AccessPoint, error) {
			return nil, ErrMckAdapterAccessPointsFailure
		}
	return adapter
}

var ErrMckAdapterSSIDFailure = errors.New("adapter SSID failure mock")

func mckAdapterSSIDFailure(t *gounit.T) *WifiAdapter {
	_, adapter := mockedDevice(t)
	adapter.dev.(*MckDevice).getPropertyAccessPoints =
		func() ([]nm.AccessPoint, error) {
			scanner.Lock()
			aa := mckAccessPoints
			scanner.Unlock()
			mck := []nm.AccessPoint{}
			for _, ap := range aa {
				mck = append(mck, &MckAccessPoint{
					AccessPoint: ap,
					getPropertySSID: func() (string, error) {
						return "", ErrMckAdapterSSIDFailure
					},
				})
			}
			return mck, nil
		}
	return adapter
}

var ErrMckAdapterSignalStrengthFailure = errors.New(
	"adapter signal strength failure mock")

func mckAdapterSignalStrengthFailure(t *gounit.T) *WifiAdapter {
	_, adapter := mockedDevice(t)
	adapter.dev.(*MckDevice).getPropertyAccessPoints =
		func() ([]nm.AccessPoint, error) {
			scanner.Lock()
			aa := mckAccessPoints
			scanner.Unlock()
			mck := []nm.AccessPoint{}
			for _, ap := range aa {
				mck = append(mck, &MckAccessPoint{
					AccessPoint: ap,
					getPropertyStrength: func() (uint8, error) {
						return 0, ErrMckAdapterSignalStrengthFailure
					},
				})
			}
			return mck, nil
		}
	return adapter
}

type MckAccessPoint struct {
	nm.AccessPoint
	getPropertySSID     func() (string, error)
	getPropertyStrength func() (uint8, error)
}

func (m *MckAccessPoint) GetPropertySSID() (string, error) {
	if m.getPropertySSID != nil {
		return m.getPropertySSID()
	}
	return m.AccessPoint.GetPropertySSID()
}

func (m *MckAccessPoint) GetPropertyStrength() (uint8, error) {
	if m.getPropertyStrength != nil {
		return m.getPropertyStrength()
	}
	return m.AccessPoint.GetPropertyStrength()
}
