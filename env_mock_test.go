package main

import (
	"errors"
	"fmt"
	"os"

	nm "github.com/Wifx/gonetworkmanager/v2"
	"github.com/godbus/dbus/v5"
	"github.com/slukits/gounit"
)

/*
NOTE this file doesn't contain any tests but mockups for Env tests.  The
_test.go suffix was added to ensure this code doesn't go into
production and doesn't need to be covered by go test -cover.
*/

const MCK_PRINT_ERR = "mocked printing error"

// mckPrintErr mocks given environment env's Lib.Println function to
// return and error.
func mckPrintErr(env *Env) *Env {
	env.Lib.Println = func(vv ...interface{}) (int, error) {
		return 0, errors.New(MCK_PRINT_ERR)
	}
	return env
}

// mckPrint mocks given environment env's Lib.Println function and
// records Println calls into given string-pointers.  mckPrint fails
// given test-instance t iff more Println calls then available string
// pointers.
func mckPrint(t *gounit.T, env *Env, pp ...*string) *Env {
	calls := 0
	env.Lib.Println = func(vv ...interface{}) (int, error) {
		if len(pp) == calls {
			t.Fatal("mock print: more Println-calls then print-pointers")
			return 0, nil
		}
		*(pp[calls]) = fmt.Sprintln(vv...)
		calls++
		return 0, nil
	}
	return env
}

// mckFatal mocks given environment env's Lib.Fatal function recording
// the fatal-message to given string-pointer print and panics with given
// panic message pnc.
func mckFatal(t *gounit.T, env *Env, pnc string, prnt *string) *Env {
	env.Lib.Fatal = func(vv ...interface{}) {
		*prnt = fmt.Sprint(vv...)
		panic(pnc)
	}
	return env
}

// mckArgs mocks up the os.Args retrieval by keeping only the first
// argument of os.Args and replacing the remaining args with given
// arguments aa.  Note if no args given the testing arguments are
// removed.
func mckArgs(env *Env, aa ...string) *Env {
	env.Lib.Args = func() []string {
		return append([]string{os.Args[0]}, aa...)
	}
	return env
}

func mckEnvVar(env *Env, name string) *Env {
	env.Lib.OsEnv = func(key string) string {
		if key == ENV_ADAPTER {
			return name
		}
		return os.Getenv(key)
	}
	return env
}

var ErrMckNewNM = errors.New("new network manager error mock")

func mckNewNMErr(env *Env) *Env {
	env.Lib.NewNM = func() (nm.NetworkManager, error) {
		return nil, ErrMckNewNM
	}
	return env
}

type NMMock struct {
	nm.NetworkManager
	allDevices             func() ([]nm.Device, error)
	failAllDevicesIfNoWifi func() ([]nm.Device, error)
}

func mockedNM(t *gounit.T) nm.NetworkManager {
	nm_, err := nm.NewNetworkManager()
	if err != nil {
		t.Fatalf("mock: nm: %v", err)
	}
	return &NMMock{NetworkManager: nm_}
}

func (m *NMMock) GetAllDevices() ([]nm.Device, error) {
	if m.failAllDevicesIfNoWifi != nil {
		m.failAllDevicesIfNoWifi()
	}
	if m.allDevices != nil {
		return m.allDevices()
	}
	return m.NetworkManager.GetAllDevices()
}

var ErrMckNMAllDevices = errors.New("nm: all devices: error mock")

func mckNMAllDevicesErr(t *gounit.T, env *Env) *Env {
	env.Lib.NewNM = func() (nm.NetworkManager, error) {
		nm_ := mockedNM(t)
		nm_.(*NMMock).allDevices = func() ([]nm.Device, error) {
			return nil, ErrMckNMAllDevices
		}
		return nm_, nil
	}
	return env
}

func mckNMDeviceType(
	t *gounit.T, env *Env, factory func(nm.Device) nm.Device,
) *Env {
	env.Lib.NewNM = func() (nm.NetworkManager, error) {
		nm_ := mockedNM(t)
		nm_.(*NMMock).allDevices = func() ([]nm.Device, error) {
			mck := []nm.Device{}
			dd, err := nm_.(*NMMock).NetworkManager.GetAllDevices()
			if err != nil {
				t.Fatalf("mock: device type error: %v", err)
				return nil, err
			}
			for _, d := range dd {
				mck = append(mck, factory(d))
			}
			return mck, nil
		}
		return nm_, nil
	}
	return env
}

type MckDeviceTypeErr struct{ nm.Device }

var ErrMckDeviceType = errors.New("device type error mock")

func (m *MckDeviceTypeErr) GetPropertyDeviceType() (
	nm.NmDeviceType, error,
) {
	return nm.NmDeviceTypeDummy, ErrMckDeviceType
}

type MckDeviceTypeNotWifi struct{ nm.Device }

func (m *MckDeviceTypeNotWifi) GetPropertyDeviceType() (
	nm.NmDeviceType, error,
) {
	return nm.NmDeviceTypeDummy, nil
}

type MckDeviceNameErr struct{ nm.Device }

func mckNMDeviceNameErr(t *gounit.T, env *Env) *Env {
	env.Lib.NewNM = func() (nm.NetworkManager, error) {
		nm_ := mockedNM(t)
		nm_.(*NMMock).allDevices = func() ([]nm.Device, error) {
			mck := []nm.Device{}
			dd, err := nm_.(*NMMock).NetworkManager.GetAllDevices()
			if err != nil {
				t.Fatalf("mock: device type error: %v", err)
				return nil, err
			}
			for _, d := range dd {
				mck = append(mck, &MckDeviceNameErr{Device: d})
			}
			return mck, nil
		}
		return nm_, nil
	}
	return env
}

var ErrMckDeviceName = errors.New("device name error mock")

func (m *MckDeviceNameErr) GetPropertyInterface() (string, error) {
	return "", ErrMckDeviceName
}

var ErrMckNewWifiDevice = errors.New("new wifi device error mock")

// mckNMWifiDeviceErr mocks also the network manager to fail provided
// test instance if there is no active wifi device
func mckNMNewWifiDeviceErr(t *gounit.T, env *Env) *Env {
	env.Lib.NewNM = func() (nm.NetworkManager, error) {
		nm_ := mockedNM(t)
		nm_.(*NMMock).allDevices = failIAllDevicesIfNoWifi(
			t, nm_.(*NMMock).NetworkManager)
		return nm_, nil
	}
	env.Lib.NewWifiDevice =
		func(op dbus.ObjectPath) (nm.DeviceWireless, error) {
			return nil, ErrMckNewWifiDevice
		}
	return env
}

type WifiDeviceStateErrMck struct{ nm.DeviceWireless }

func mckNMWifiDeviceStateErr(t *gounit.T, env *Env) *Env {
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
			return &WifiDeviceStateErrMck{DeviceWireless: wd}, nil
		}
	return env
}

var ErrMckWifiDeviceState = errors.New("wifi device state err mock")

func (m *WifiDeviceStateErrMck) GetPropertyState() (
	nm.NmDeviceState, error,
) {
	return nm.NmDeviceStateUnknown, ErrMckWifiDeviceState
}

type MckInactiveWifiDeviceState struct{ nm.DeviceWireless }

func mckNMInactiveWifiDevice(t *gounit.T, env *Env) *Env {
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
			return &MckInactiveWifiDeviceState{DeviceWireless: wd}, nil
		}
	return env
}

func (m *MckInactiveWifiDeviceState) GetPropertyState() (
	nm.NmDeviceState, error,
) {
	return nm.NmDeviceStateUnavailable, nil
}

type WifiDeviceNameErrMck struct{ nm.DeviceWireless }

func mckNMWifiDeviceNameErr(t *gounit.T, env *Env) *Env {
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
			return &WifiDeviceNameErrMck{DeviceWireless: wd}, nil
		}
	return env
}

var ErrMckWifiDeviceName = errors.New("wifi device name err mock")

func (m *WifiDeviceNameErrMck) GetPropertyInterface() (string, error) {
	return "", ErrMckWifiDeviceName
}

// failIAllDevicesIfNoWifi itself should be probably tested :)
func failIAllDevicesIfNoWifi(t *gounit.T, nm_ nm.NetworkManager) func() ([]nm.Device, error) {
	return func() ([]nm.Device, error) {
		fatalStr := "mock: fail all devices if no wifi: %v"
		dd, err := nm_.GetAllDevices()
		if err != nil {
			t.Fatalf(fatalStr, err)
			return nil, err
		}
		for _, d := range dd {
			type_, err := d.GetPropertyDeviceType()
			if err != nil {
				t.Fatalf(fatalStr, err)
				return nil, err
			}
			if type_ != nm.NmDeviceTypeWifi {
				continue
			}
			wd, err := nm.NewDeviceWireless(d.GetPath())
			if err != nil {
				t.Fatalf(fatalStr, err)
				return nil, err
			}
			state, err := wd.GetPropertyState()
			if err != nil {
				t.Fatalf(fatalStr, err)
				return nil, err
			}
			if state != nm.NmDeviceStateActivated &&
				state != nm.NmDeviceStateDisconnected {
				continue
			}
			return dd, nil
		}
		return nil, fmt.Errorf(fatalStr, "no active wifi device")
	}
}
