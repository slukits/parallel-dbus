package main

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	nm "github.com/Wifx/gonetworkmanager/v2"
	"github.com/godbus/dbus/v5"
	. "github.com/slukits/gounit"
)

type RequestHandler struct{ Suite }

func (s *RequestHandler) SetUp(t *T) { t.Parallel() }

func (s *RequestHandler) Prints_help_if_no_sub_command_given(t *T) {
	got := ""
	handleRequest(mckArgs(mckPrint(t, &Env{}, &got)))
	t.Contains(got, help)
}

func (s *RequestHandler) Fails_on_unknown_sub_command(t *T) {
	expPnc, expErr, subCmd := "fatal mock panic", "", "unknown"
	defer func() {
		t.Eq(expPnc, recover().(string))
		t.Contains(expErr, fmt.Sprintf(subErr, subCmd))
	}()
	handleRequest(mckArgs(
		mckFatal(t, &Env{}, expPnc, &expErr), subCmd))
}

func (s *RequestHandler) Fails_on_failing_device_retrieval(t *T) {
	expPnc, expErr := "fatal mock panic", ""
	defer func() {
		t.Eq(expPnc, recover().(string))
		t.Contains(expErr, ErrDeviceNotFound.Error())
	}()
	handleRequest(mckEnvVar(mckArgs(
		mckFatal(t, &Env{}, expPnc, &expErr)), "unknown"))
}

type MckDeviceScanFailing struct {
	nm.DeviceWireless
}

func (s *RequestHandler) Fails_on_failing_scan(t *T) {
	env := &Env{}
	env.Lib.NewWifiDevice =
		func(op dbus.ObjectPath) (nm.DeviceWireless, error) {
			wd, err := nm.NewDeviceWireless(op)
			t.FatalOn(err)
			return &MckDeviceScanFailing{DeviceWireless: wd}, nil
		}
	expPnc, expErr := "fatal mock panic", ""
	defer func() {
		t.Eq(expPnc, recover().(string))
		t.Contains(expErr, ErrMckDeviceScanFailing.Error())
	}()
	handleRequest(mckArgs(
		mckFatal(t, env, expPnc, &expErr), "scan"))
}

func (s *RequestHandler) Prints_available_SSID_on_scan(t *T) {
	env := mckDevices(t)
	out := []string{}
	env.Lib.Println = func(vv ...interface{}) (int, error) {
		out = append(out, vv[0].(string))
		return 0, nil
	}
	handleRequest(mckArgs(env, "scan"))
	t.Eq(len(out), len(mckAccessPoints))
	got := strings.Join(out, "\n")
	for _, ap := range mckAccessPoints {
		SSID, err := ap.GetPropertySSID()
		t.FatalOn(err)
		t.Contains(got, SSID)
	}
}

var ErrMckDeviceScanFailing = errors.New("device scan failing mock")

func (m *MckDeviceScanFailing) RequestScan() error {
	return ErrMckDeviceScanFailing
}

func TestRequestHandler(t *testing.T) {
	t.Parallel()
	Run(&RequestHandler{}, t)
}
