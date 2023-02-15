package main

import "fmt"

const help = `
NAME

wifi - a simple commandline client to scan for, connect to and 
         disconnect from wifi SSIDs.

SYNOPSIS

	wifi scan|connect SSID|disconnect [--wifi-adapter='DEVICE-NAME']


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

	scan	provides all SSIDs and their signal strength which can
		be reached by a given wifi-adapter.

	connect SSID
		connects to given SSID at given adapter querying a password
		if the wifi-network is not open.

	disconnect
		closes the current connection at given adapter.


COMMAND LINE OPTIONS

	--wifi-adapter='DEVICE-NAME'
		lets you set the used wifi-adapter e.g.:

			$ wifi scan --wifi-adapter='wlan0'

		Note the --wifi-adapter option overwrites a set WIFI_ADAPTER 
		environment variable.
`

const subErr = `
wifi: error: unknown sub-command: '%s'
call wifi without any argument to see its help.
`

const deviceErr = `
wifi: error: device retrieval: %v
call wifi without any argument to see its help.
`

func handleRequest(env *Env) {
	dev, err := env.Device()
	if err != nil {
		env.Fatal(fmt.Sprintf(deviceErr, err))
	}
	_ = dev
	switch env.Sub() {
	case ZeroSub:
		env.Println(help)
	default:
		env.Fatal(fmt.Sprintf(subErr, env.Sub()))
	}
}

func main() { handleRequest(&Env{}) }
