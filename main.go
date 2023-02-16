package main

import "fmt"

const help = `
NAME

wifi - a simple commandline client to scan for, connect to and 
       disconnect from wifi access points reachable by a given
       wifi-adapter.

SYNOPSIS

	wifi active|scan|disconnect|connect SSID|delete SSID 
		[--wifi-adapter='DEVICE-NAME']


DESCRIPTION

	If wifi is called without any arguments this help is shown.
	Only one subcommand at a time may be used.  The used wifi adapter 
	for the subcommands defaults to the first found active wifi adapter
	if neither an environment variable is set nor an adapter option is
	given.  If an according environment variable is set, e.g.:

		$ WIFI_ADAPTER=wlan0 wifi scan

	then *wifi* tries to use this adapter.  A set adapter commandline 
	option (see below) supersedes an environment variable.


SUBCOMMANDS

	active	provides the SSID of the active wifi connection.

	scan	provides all SSIDs and their signal strength which can
		be reached by a given wifi-adapter.

	disconnect
		closes the current connection at given adapter.

	connect SSID
		connects to given SSID at given adapter querying a password
		if the access point with given SSID is not configured.

	delete SSID
		deletes the configuration of the wifi access point with
		given SSID


COMMAND LINE OPTIONS

	--wifi-adapter='DEVICE-NAME'
		lets you set the used wifi-adapter e.g.:

			$ wifi scan --wifi-adapter='wlan0'

		Note the --wifi-adapter option overwrites a set WIFI_ADAPTER 
		environment variable and the single-quotes may not be omitted.
`

const subErr = `
wifi: error: unknown sub-command: '%s'
call wifi without any argument to see its help.
`

const deviceErr = `
wifi: error: device retrieval: %v
call wifi without any argument to see its help.
`

const scanErr = `
wifi: error: scan '%s': %v
call wifi without any argument to see its help.
`

const disconnectErr = `
wifi: error: disconnect '%s': %v
call wifi without any argument to see its help.
`

const activeErr = `
wifi: error: active on '%s': %v
call wifi without any argument to see its help.
`

const connectErr = `
wifi: error: connect on '%s': %v
call wifi without any argument to see its help.
`

const delErr = `
wifi: error: delete on '%s': %v
call wifi without any argument to see its help.
`

func handleRequest(env *Env) {
	dev, err := env.Device()
	if err != nil {
		env.Fatal(fmt.Sprintf(deviceErr, err))
	}
	switch env.Sub() {
	case ActiveSub:
		ssid, err := dev.Active()
		if err != nil {
			env.Fatal(fmt.Sprintf(activeErr, dev.Name(), err))
		}
		env.Println(fmt.Sprintf("active access point on '%s' is: '%s'",
			dev.Name(), ssid))
	case ScanSub:
		aa, err := dev.Scan()
		if err != nil {
			env.Fatal(fmt.Sprintf(scanErr, dev.Name(), err))
		}
		for _, a := range aa {
			env.Println(fmt.Sprintf(
				"SSID: %s, strength: %d", a.SSID, a.Strength))
		}
	case DisconnectSub:
		if err := dev.Disconnect(); err != nil {
			env.Fatal(fmt.Sprintf(disconnectErr, dev.Name(), err))
		}
	case ConnectSub:
		ssid := env.SSID()
		if ssid == "" {
			env.Fatal(fmt.Sprintf(connectErr, dev.Name(),
				"missing SSID"))
		}
		if err := dev.Connect(ssid); err != nil {
			env.Fatal(fmt.Sprintf(connectErr, dev.Name(), err))
		}
	case DeleteSub:
		ssid := env.SSID()
		if ssid == "" {
			env.Fatal(fmt.Sprintf(delErr, dev.Name(), "missing SSID"))
		}
		if err := dev.Delete(ssid); err != nil {
			env.Fatal(fmt.Sprintf(delErr, dev.Name(), err))
		}
	case ZeroSub:
		env.Println(help)
	default:
		env.Fatal(fmt.Sprintf(subErr, env.Sub()))
	}
}

func main() { handleRequest(&Env{}) }
