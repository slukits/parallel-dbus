package main

import (
	"errors"
	"fmt"
	"os"

	nm "github.com/Wifx/gonetworkmanager/v2"
)

func firstWifiDevice(m nm.NetworkManager) (nm.Device, error) {
	devices, err := m.GetPropertyAllDevices()
	if err != nil {
		return nil, err
	}
	for _, d := range devices {
		type_, err := d.GetPropertyDeviceType()
		if err != nil {
			return nil, err
		}
		if type_ != nm.NmDeviceTypeWifi {
			continue
		}
		return d, nil
	}
	return nil, errors.New("no wifi device found")
}

func main() {

	/* Create new instance of gonetworkmanager */
	m, err := nm.NewNetworkManager()
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	wifi, err := firstWifiDevice(m)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	deviceInterface, err := wifi.GetPropertyInterface()
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	fmt.Println(deviceInterface + " - " + string(wifi.GetPath()))

	wifiDev, err := nm.NewDeviceWireless(wifi.GetPath())
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	state, err := wifiDev.GetPropertyState()
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	fmt.Println(state.String())

	/* Get devices */
	// devices, err := m.GetPropertyAllDevices()
	// if err != nil {
	// 	fmt.Println(err.Error())
	// 	os.Exit(1)
	// }

	/* Show each device path and interface name */
	// for _, device := range devices {

	// 	deviceInterface, err := device.GetPropertyInterface()
	// 	if err != nil {
	// 		fmt.Println(err.Error())
	// 		continue
	// 	}
	// 	dtype, err := device.GetPropertyDeviceType()
	// 	if err != nil {
	// 		fmt.Println(err.Error())
	// 		continue
	// 	}
	// 	prefix := "no-wifi: "
	// 	if dtype == nm.NmDeviceTypeWifi {
	// 		prefix = "wifi: "
	// 	}

	// 	fmt.Println(prefix + deviceInterface + " - " + string(device.GetPath()))
	// }

	os.Exit(0)
}
