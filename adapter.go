package main

import nm "github.com/Wifx/gonetworkmanager/v2"

type WifiAdapter struct {
	name string
	dev  nm.DeviceWireless
}

func (a *WifiAdapter) IsActivated() bool {
	state, err := a.dev.GetPropertyState()
	if err != nil {
		return false
	}
	return state == nm.NmDeviceStateActivated
}

func (a *WifiAdapter) DeviceName() string { return a.name }
