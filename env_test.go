package main

import (
	"fmt"
	"testing"

	nm "github.com/Wifx/gonetworkmanager/v2"
	. "github.com/slukits/gounit"
)

type AnEnv struct{ Suite }

func (s *AnEnv) SetUp(t *T) { t.Parallel() }

func (s *AnEnv) Panics_if_lib_s_fatal_call_doesnt_end_execution(t *T) {
	env := &Env{}
	env.Lib.Fatal = func(vv ...interface{}) {}
	t.Panics(func() { env.Fatal("") })
}

func (s *AnEnv) Dies_if_lib_s_print_line_fails(t *T) {
	died, exitMock := false, "execution end mock"
	env := &Env{}
	env.Lib.Fatal = func(vv ...interface{}) {
		died = true
		t.Eq(vv[0].(error).Error(), MCK_PRINT_ERR)
		panic(exitMock)
	}
	mckPrintErr(env)
	defer func() {
		t.Eq(recover().(string), exitMock)
		t.True(died)
	}()
	env.Println("failing print")
}

func (s *AnEnv) Returns_the_zero_sub_command_on_no_cmd_args(t *T) {
	t.Eq(mckArgs(&Env{}).Sub(), ZeroSub)
}

func (s *AnEnv) Default_Device_fails_if_NM_cant_be_obtained(t *T) {
	_, err := mckNewNMErr(mckArgs(&Env{})).Device()
	t.ErrIs(err, ErrNewNM)
	t.ErrIs(err, ErrMckNewNM)
}

func (s *AnEnv) Default_Device_fails_if_devices_cant_be_obtained(t *T) {
	_, err := mckNMAllDevicesErr(t, mckArgs(&Env{})).Device()
	t.ErrIs(err, ErrNMAllDevices)
	t.ErrIs(err, ErrMckNMAllDevices)
}

func (s *AnEnv) Default_device_fails_if_device_type_unobtainable(t *T) {
	factory := func(d nm.Device) nm.Device {
		return &MckDeviceTypeErr{Device: d}
	}
	_, err := mckNMDeviceType(t, mckArgs(&Env{}), factory).Device()
	t.ErrIs(err, ErrDeviceType)
	t.ErrIs(err, ErrMckDeviceType)
}

func (s *AnEnv) Default_device_fails_if_wifi_device_creation_fails(
	t *T,
) {
	_, err := mckNMNewWifiDeviceErr(t, mckArgs(&Env{})).Device()
	t.ErrIs(err, ErrNewWifiDevice)
	t.ErrIs(err, ErrMckNewWifiDevice)
}

func (s *AnEnv) Default_device_fails_if_device_state_fails(t *T) {
	_, err := mckNMWifiDeviceStateErr(t, mckArgs(&Env{})).Device()
	t.ErrIs(err, ErrWifiDeviceState)
	t.ErrIs(err, ErrMckWifiDeviceState)
}

func (s *AnEnv) Default_device_fails_if_device_name_fails(t *T) {
	_, err := mckNMWifiDeviceNameErr(t, mckArgs(&Env{})).Device()
	t.ErrIs(err, ErrDeviceName)
	t.ErrIs(err, ErrMckWifiDeviceName)
}

func (s *AnEnv) Default_device_fails_if_no_active_device_found(t *T) {
	_, err := mckNMInactiveWifiDevice(t, mckArgs(&Env{})).Device()
	t.ErrIs(err, ErrWifiDevice)
}

func (s *AnEnv) Provides_an_active_wifi_device_by_default(t *T) {
	wd, err := (mckArgs(&Env{})).Device()
	t.FatalOn(err)
	t.True(wd.IsActivated())
}

func (s *AnEnv) Provides_named_device_from_command_line_argument(t *T) {
	wd, err := (mckArgs(&Env{})).Device()
	t.FatalOn(err)
	arg := fmt.Sprintf("%s%s'", ADAPTER_PREFIX, wd.DeviceName())
	// NOTE we can only see by coverage that Device() takes a different
	// execution path with arg than without arg
	argWD, err := mckArgs(&Env{}, arg).Device()
	t.FatalOn(err)
	t.Eq(argWD.DeviceName(), wd.DeviceName())
}

func (s *AnEnv) Provides_named_device_from_env_variable(t *T) {
	wd, err := (mckArgs(&Env{})).Device()
	t.FatalOn(err)
	varWD, err := mckEnvVar(mckArgs(&Env{}), wd.DeviceName()).Device()
	t.FatalOn(err)
	t.Eq(wd.DeviceName(), varWD.DeviceName())
	// cover commandline argument path with more than one argument
	wd, err = (mckArgs(&Env{}, "some-arg")).Device()
	t.FatalOn(err)
	varWD, err = mckEnvVar(mckArgs(&Env{}), wd.DeviceName()).Device()
	t.FatalOn(err)
	t.Eq(wd.DeviceName(), varWD.DeviceName())
}

func (s *AnEnv) Named_Device_fails_if_NM_unobtainable(t *T) {
	wd, err := (mckArgs(&Env{})).Device()
	t.FatalOn(err)
	_, err = mckNewNMErr(mckEnvVar(mckArgs(
		&Env{}), wd.DeviceName())).Device()
	t.ErrIs(err, ErrNewNM)
	t.ErrIs(err, ErrMckNewNM)
}

func (s *AnEnv) Named_Device_fails_if_devices_unobtainable(t *T) {
	wd, err := (mckArgs(&Env{})).Device()
	t.FatalOn(err)
	_, err = mckNMAllDevicesErr(t, mckEnvVar(mckArgs(
		&Env{}), wd.DeviceName())).Device()
	t.ErrIs(err, ErrNMAllDevices)
	t.ErrIs(err, ErrMckNMAllDevices)
}

func (s *AnEnv) Named_Device_fails_if_device_name_retrieval_fails(t *T) {
	wd, err := (mckArgs(&Env{})).Device()
	t.FatalOn(err)
	_, err = mckNMDeviceNameErr(t, mckEnvVar(mckArgs(
		&Env{}), wd.DeviceName())).Device()
	t.ErrIs(err, ErrDeviceName)
	t.ErrIs(err, ErrMckDeviceName)
}

func (s *AnEnv) Named_Device_fails_if_device_type_retrieval_fails(t *T) {
	wd, err := (mckArgs(&Env{})).Device()
	t.FatalOn(err)
	factory := func(d nm.Device) nm.Device {
		return &MckDeviceTypeErr{Device: d}
	}
	_, err = mckNMDeviceType(t, mckEnvVar(mckArgs(
		&Env{}), wd.DeviceName()), factory).Device()
	t.ErrIs(err, ErrDeviceType)
	t.ErrIs(err, ErrMckDeviceType)
}

func (s *AnEnv) Named_Device_fails_if_its_type_is_not_wifi(t *T) {
	wd, err := (mckArgs(&Env{})).Device()
	t.FatalOn(err)
	factory := func(d nm.Device) nm.Device {
		return &MckDeviceTypeNotWifi{Device: d}
	}
	_, err = mckNMDeviceType(t, mckEnvVar(mckArgs(
		&Env{}), wd.DeviceName()), factory).Device()
	t.ErrIs(err, ErrWifiDevice)
	t.ErrIs(err, ErrNoWifi)
}

func (s *AnEnv) Named_Device_fails_if_wifi_device_creation_fails(t *T) {
	wd, err := (mckArgs(&Env{})).Device()
	t.FatalOn(err)
	_, err = mckNMNewWifiDeviceErr(t, mckEnvVar(mckArgs(
		&Env{}), wd.DeviceName())).Device()
	t.ErrIs(err, ErrNewWifiDevice)
	t.ErrIs(err, ErrMckNewWifiDevice)
}

func (s *AnEnv) Named_Device_fails_if_state_retrieval_fails(t *T) {
	wd, err := (mckArgs(&Env{})).Device()
	t.FatalOn(err)
	_, err = mckNMWifiDeviceStateErr(t, mckEnvVar(mckArgs(
		&Env{}), wd.DeviceName())).Device()
	t.ErrIs(err, ErrWifiDeviceState)
	t.ErrIs(err, ErrMckWifiDeviceState)
}

func (s *AnEnv) Named_Device_fails_if_state_not_activated(t *T) {
	wd, err := (mckArgs(&Env{})).Device()
	t.FatalOn(err)
	_, err = mckNMInactiveWifiDevice(t, mckEnvVar(mckArgs(
		&Env{}), wd.DeviceName())).Device()
	t.ErrIs(err, ErrWifiDevice)
	t.ErrIs(err, ErrNotActivated)
}

func (s *AnEnv) Named_Device_fails_if_device_unknown(t *T) {
	_, err := mckEnvVar(mckArgs(&Env{}), "unknown").Device()
	t.ErrIs(err, ErrWifiDevice)
	t.ErrIs(err, ErrDeviceNotFound)
}

func TestAnEnv(t *testing.T) {
	t.Parallel()
	Run(&AnEnv{}, t)
}
