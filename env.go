package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	nm "github.com/Wifx/gonetworkmanager/v2"
	"github.com/godbus/dbus/v5"
)

// Env main purpose is to provide a mockable system environment and a
// wifi device (see [Env.Device]) from the NetworkManager dbus API
// according to an optional given commandline argument or an optional
// set environment variable.
type Env struct {

	// Lib provides library functions which may fail or exit execution,
	// e.g. fmt.Println, log.Fatal or nm.NewNetworkManager.
	Lib EnvLib

	// libInit indicates if Lib-property has been set to its defaults
	// where not mocked.  NOTE the Lib-property is set only once to its
	// default.
	libInit bool

	// _nm create only one network-manager instance per Env
	_nm nm.NetworkManager
}

// lib set the defaults for library functions and system environment
// evaluation.
func (e *Env) lib() EnvLib {
	if !e.libInit {
		e.libInit = true
		if e.Lib.Println == nil {
			e.Lib.Println = fmt.Println
		}
		if e.Lib.Fatal == nil {
			e.Lib.Fatal = log.Fatal
		}
		if e.Lib.Args == nil {
			e.Lib.Args = e.args
		}
		if e.Lib.OsEnv == nil {
			e.Lib.OsEnv = os.Getenv
		}
		if e.Lib.NewNM == nil {
			e.Lib.NewNM = nm.NewNetworkManager
		}
		if e.Lib.NewWifiDevice == nil {
			e.Lib.NewWifiDevice = nm.NewDeviceWireless
		}
		if e.Lib.NewWifiAdapter == nil {
			e.Lib.NewWifiAdapter = e.newWifiAdapter
		}
	}
	return e.Lib
}

func (e *Env) args() []string { return os.Args }

// Println prints given values vv to given environment e's standard
// library printer.
func (e *Env) Println(vv ...interface{}) {
	if _, err := e.lib().Println(vv...); err != nil {
		e.lib().Fatal(err)
	}
}

// Fatal passes given values vv to given environment e's standard
// library fatal-er.
func (e *Env) Fatal(vv ...interface{}) {
	e.lib().Fatal(vv...)
	panic("env: expected execution to end")
}

// Sub returns potentially given sub-command, that is the second
// argument, if one exists otherwise the zero-string.
func (e *Env) Sub() SubCommand {
	args := e.lib().Args()
	if len(args) < 2 {
		return ZeroSub
	}
	return SubCommand(args[1])
}

// ADAPTER_PREFIX is the prefix of the adapter commandline argument
const ADAPTER_PREFIX = "--wifi-adapter='"

// ENV_ADAPTER is the name of the adapter environment variable
const ENV_ADAPTER = "WIFI_ADAPTER"

var ErrEnvDevice = errors.New("env: determine device")

// Device evaluates the program arguments, environment variables and the
// NetworkManager to determine a wifi-adapter and returns it;  Device
// fails if no active or disconnected wifi-adapter is found.  Device
// evaluates all possible options in the following order:
//   - if the last commandline argument has the ADAPTER_PREFIX Env tries
//     to use this adapter and fails if something goes wrong
//   - is no commandline argument given Env checks for the ENV_ADAPTER os
//     environment variable and tries to use set value failing if given
//     name is not an active wifi device
//   - is also no environment variable given WifiAdapter defaults to the
//     first active or disconnected wifi-adapter which can be obtained from the
//     NetworkManager
func (e *Env) Device() (*WifiAdapter, error) {
	adapter, err := e.argDevice()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrEnvDevice, err)
	}
	if adapter != nil {
		return adapter, nil
	}
	adapter, err = e.envDevice()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrEnvDevice, err)
	}
	if adapter != nil {
		return adapter, nil
	}
	adapter, err = e.defaultDevice()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrEnvDevice, err)
	}
	return adapter, nil
}

// SSID returns given environment e's SSID commandline argument which is
// the third argument if set or the zero string otherwise.
func (e *Env) SSID() string {
	if len(e.lib().Args()) < 3 {
		return ""
	}
	return e.lib().Args()[2]
}

func (e *Env) argDevice() (*WifiAdapter, error) {
	if len(e.lib().Args()) < 2 {
		return nil, nil
	}
	aa := e.lib().Args()
	if !strings.HasPrefix(aa[len(aa)-1], ADAPTER_PREFIX) {
		return nil, nil
	}
	name := strings.TrimPrefix(aa[len(aa)-1], ADAPTER_PREFIX)
	name = strings.TrimSuffix(name, "'")
	return e.namedDevice(name)
}

func (e *Env) envDevice() (*WifiAdapter, error) {
	name := e.lib().OsEnv(ENV_ADAPTER)
	if name == "" {
		return nil, nil
	}
	return e.namedDevice(name)
}

func (e *Env) namedDevice(name string) (*WifiAdapter, error) {
	nm_, err := e.nm()
	if err != nil {
		return nil, err
	}
	dd, err := nm_.GetAllDevices()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrNMAllDevices, err)
	}
	for _, d := range dd {
		name_, err := d.GetPropertyInterface()
		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrDeviceName, err)
		}
		if name != name_ {
			continue
		}
		type_, err := d.GetPropertyDeviceType()
		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrDeviceType, err)
		}
		if type_ != nm.NmDeviceTypeWifi {
			return nil, fmt.Errorf("%w: '%s' is %w",
				ErrWifiDevice, name, ErrNoWifi)
		}
		wd, err := e.lib().NewWifiDevice(d.GetPath())
		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrNewWifiDevice, err)
		}
		state, err := wd.GetPropertyState()
		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrWifiDeviceState, err)
		}
		if state != nm.NmDeviceStateActivated &&
			state != nm.NmDeviceStateDisconnected {
			return nil, fmt.Errorf("%w: '%s' %w",
				ErrWifiDevice, name, ErrNotActivated)
		}
		return e.lib().NewWifiAdapter(wd, name), nil
	}
	return nil, fmt.Errorf("%w: '%s' %w",
		ErrWifiDevice, name, ErrDeviceNotFound)
}

var ErrNoWifi = errors.New("no wifi device")
var ErrDeviceNotFound = errors.New("device not found")
var ErrNotActivated = errors.New("not activated")

func (e *Env) defaultDevice() (*WifiAdapter, error) {
	nm_, err := e.nm()
	if err != nil {
		return nil, err
	}
	dd, err := nm_.GetAllDevices()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrNMAllDevices, err)
	}
	for _, d := range dd {
		type_, err := d.GetPropertyDeviceType()
		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrDeviceType, err)
		}
		if type_ != nm.NmDeviceTypeWifi {
			continue
		}
		wd, err := e.lib().NewWifiDevice(d.GetPath())
		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrNewWifiDevice, err)
		}
		state, err := wd.GetPropertyState()
		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrWifiDeviceState, err)
		}
		if state != nm.NmDeviceStateActivated &&
			state != nm.NmDeviceStateDisconnected {
			continue
		}
		name, err := wd.GetPropertyInterface()
		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrDeviceName, err)
		}
		return e.lib().NewWifiAdapter(wd, name), nil
	}
	return nil, fmt.Errorf("%w: %s", ErrWifiDevice,
		"no active wifi adapter")
}

func (e *Env) newWifiAdapter(
	d nm.DeviceWireless, n string,
) *WifiAdapter {
	return &WifiAdapter{
		Timeout: 10 * time.Second,
		name:    n,
		dev:     d,
		env:     e,
	}
}

var ErrDeviceName = errors.New("env: device: name")
var ErrWifiDevice = errors.New("env: device")
var ErrWifiDeviceState = errors.New("env: device: state")
var ErrNewWifiDevice = errors.New("env: new device")
var ErrDeviceType = errors.New("env: device type")
var ErrNMAllDevices = errors.New("env: all devices")
var ErrNewNM = errors.New("env: new network manager")

func (e *Env) nm() (nm.NetworkManager, error) {
	if e._nm == nil {
		nm_, err := e.lib().NewNM()
		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrNewNM, err)
		}
		e._nm = nm_
	}
	return e._nm, nil
}

// EnvLib provides standard library functions and system-properties which
// should be mockable to simplify testing.
type EnvLib struct {

	// Println defaults to fmt.Println
	Println func(vv ...interface{}) (int, error)

	// Fatal defaults to log.Fatal
	Fatal func(vv ...interface{})

	// Args defaults to func() []string { return os.Args }
	Args func() []string

	// OsEnv defaults to os.Getenv
	OsEnv func(string) string

	// NewNM defaults to gonetworkmanager.NewNetworkManager
	NewNM func() (nm.NetworkManager, error)

	// NewWifiDevice defaults to gonetworkmanager.NewDeviceWireless
	NewWifiDevice func(dbus.ObjectPath) (nm.DeviceWireless, error)

	// NewWifiAdapter defaults to Env.newWifiAdapter
	NewWifiAdapter func(nm.DeviceWireless, string) *WifiAdapter
}

type SubCommand string

const (
	ZeroSub       SubCommand = ""
	ScanSub       SubCommand = "scan"
	ConnectSub    SubCommand = "connect"
	DisconnectSub SubCommand = "disconnect"
	ActiveSub     SubCommand = "active"
	DeleteSub     SubCommand = "delete"
)
