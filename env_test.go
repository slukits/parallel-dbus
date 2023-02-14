package main

import (
	"fmt"
	"log"
	"os"
	"testing"

	nm "github.com/Wifx/gonetworkmanager/v2"
	. "github.com/slukits/gounit"
)

type AnEnv struct{ Suite }

func (s *AnEnv) SetUp(t *T) { t.Parallel() }

// Initializes_its_std_lib_defaults is the only white-box test using the
// private Env.lib implementation detail.
func (s *AnEnv) Initializes_its_library_defaults(t *T) {
	env := &Env{}
	lib := env.lib()
	t.Eq(
		fmt.Sprintf("%T::%[1]p", fmt.Println),
		fmt.Sprintf("%T::%[1]p", lib.Println),
	)
	t.Eq(
		fmt.Sprintf("%T::%[1]p", log.Fatal),
		fmt.Sprintf("%T::%[1]p", lib.Fatal),
	)
	t.Eq(
		fmt.Sprintf("%T::%[1]p", env.args),
		fmt.Sprintf("%T::%[1]p", lib.Args),
	)
	t.Eq(
		fmt.Sprintf("%T::%[1]p", os.Getenv),
		fmt.Sprintf("%T::%[1]p", lib.OsEnv),
	)
	t.Eq(
		fmt.Sprintf("%T::%[1]p", nm.NewNetworkManager),
		fmt.Sprintf("%T::%[1]p", lib.NewNM),
	)
	t.Eq(
		fmt.Sprintf("%T::%[1]p", nm.NewDeviceWireless),
		fmt.Sprintf("%T::%[1]p", lib.NewWifiDevice),
	)
}

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
	_, err := mckNMDeviceTypeErr(t, mckArgs(&Env{})).Device()
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
	_, err := mckNMDeviceNameErr(t, mckArgs(&Env{})).Device()
	t.ErrIs(err, ErrDeviceName)
	t.ErrIs(err, ErrMckDeviceName)
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
	// NOTE we can only see by coverage that Device() takes a different
	// execution path with arg than without arg
	varWD, err := mckEnvVar(mckArgs(&Env{}), wd.DeviceName()).Device()
	t.FatalOn(err)
	t.Eq(wd.DeviceName(), varWD.DeviceName())
}

func TestAnEnv(t *testing.T) {
	t.Parallel()
	Run(&AnEnv{}, t)
}
