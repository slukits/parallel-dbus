package main

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"time"

	nm "github.com/Wifx/gonetworkmanager/v2"
	"github.com/godbus/dbus/v5"
	"github.com/google/uuid"
	"golang.org/x/term"
)

// BusConnection reduces the dbus.Conn-API to the needs of WifiAdapter
// and makes these features mockable
type BusConnection interface {
	AddMatchSignal(...dbus.MatchOption) error
	Signal(chan<- *dbus.Signal)
	RemoveSignal(chan<- *dbus.Signal)
	Close() error
}

// WifiAdapter wraps a wireless device of network-manager and provides a
// convenient API to scan, connect and disconnect to wifi access points.
type WifiAdapter struct {

	// Lib provide mockable library features
	Lib AdapterLib

	// Timeout determines how long an adapter operation waits for an
	// expected signal to occur.
	Timeout time.Duration

	name    string
	dev     nm.DeviceWireless
	env     *Env
	libInit bool
}

func (a *WifiAdapter) lib() AdapterLib {
	if !a.libInit {
		a.libInit = true
		if a.Lib.SystemBus == nil {
			a.Lib.SystemBus = func() (BusConnection, error) {
				return dbus.ConnectSystemBus()
			}
		}
		if a.Lib.WaitForPropertyChange == nil {
			a.Lib.WaitForPropertyChange = a.waitForPropertyChange
		}
		if a.Lib.Disconnect == nil {
			a.Lib.Disconnect = a.dev.Disconnect
		}
	}
	return a.Lib
}

// IsActivated returns true if given wifi-adapter is in an activated
// state, i.e. connected to an wifi access point.
func (a *WifiAdapter) IsActivated() bool {
	state, err := a.dev.GetPropertyState()
	if err != nil {
		return false
	}
	return state == nm.NmDeviceStateActivated
}

var ErrAdapterScan = errors.New("wifi-adapter: scan error")

// AccessPoint provides the SSID and signal strength of an wifi access
// point.
type AccessPoint struct {
	SSID     string
	Strength uint8
}

// Scan for all available access points of given wifi-adapter a and
// return found access points sorted descending by signal strength and
// ascending by SSID.
func (a *WifiAdapter) Scan() (_ []AccessPoint, err error) {
	c, dfr, err := a.setupSignalMatcher()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrAdapterScan, err)
	}
	defer func() { err = dfr(err, ErrAdapterScan) }()
	if err := a.dev.RequestScan(); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrAdapterScan, err)
	}
	if err := a.lib().WaitForPropertyChange(c, "LastScan"); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrAdapterScan, err)
	}
	aa, err := a.dev.GetPropertyAccessPoints()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrAdapterScan, err)
	}
	accessPoints := []AccessPoint{}
	for _, ap := range aa {
		ssid, err := ap.GetPropertySSID()
		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrAdapterScan, err)
		}
		strength, err := ap.GetPropertyStrength()
		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrAdapterScan, err)
		}
		accessPoints = append(accessPoints, AccessPoint{
			SSID: ssid, Strength: strength})
	}
	sort.Slice(accessPoints, func(i, j int) bool {
		return accessPoints[i].SSID < accessPoints[j].SSID
	})
	sort.SliceStable(accessPoints, func(i, j int) bool {
		return accessPoints[i].Strength > accessPoints[j].Strength
	})
	return accessPoints, nil
}

// Disconnect currently active access point of given wife-adapter a.
func (a *WifiAdapter) Disconnect() (err error) {
	if !a.IsActivated() {
		return errors.New("not connected")
	}
	c, dfr, err := a.setupSignalMatcher()
	if err != nil {
		return fmt.Errorf("%w: %w", ErrAdapterScan, err)
	}
	defer func() { err = dfr(err, ErrAdapterScan) }()
	if err := a.lib().Disconnect(); err != nil {
		return err
	}
	return a.waitForStateChange(c, nm.NmDeviceStateDisconnected)
}

// Active returns the SSID of given wifi adapter a's connected access
// point.
func (a *WifiAdapter) Active() (string, error) {
	ap, err := a.dev.GetPropertyActiveAccessPoint()
	if err != nil {
		return "", err
	}
	ssid, err := ap.GetPropertySSID()
	if err != nil {
		return "", err
	}
	return ssid, nil
}

var ErrAdapterConnect = errors.New("wifi-adapter: connecting error")

// Connect to the given wifi-adapter a to the access-point with given
// SSID.  If no configuration settings for SSID found query a password,
// create new configuration settings for SSID and connect.
func (a *WifiAdapter) Connect(SSID string) error {
	cnn, err := a.settingsConnectionOf(SSID)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrAdapterConnect, err)
	}
	c, dfr, err := a.setupSignalMatcher()
	if err != nil {
		return fmt.Errorf("%w: %w", ErrAdapterConnect, err)
	}
	defer func() { err = dfr(err, ErrAdapterConnect) }()
	if cnn != nil {
		if err := a.activateKnownAccessPoint(cnn, SSID); err != nil {
			return fmt.Errorf("%w: %w", ErrAdapterConnect, err)
		}
		err := a.waitForStateChange(c, nm.NmDeviceStateActivated)
		if err != nil {
			return fmt.Errorf("%w: %w", ErrAdapterConnect, err)
		}
		return nil
	}
	if err := a.configureNewConnection(SSID); err != nil {
		return fmt.Errorf("%w: %w", ErrAdapterConnect, err)
	}
	err = a.waitForStateChange(c, nm.NmDeviceStateActivated)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrAdapterConnect, err)
	}
	return nil
}

func (a *WifiAdapter) configureNewConnection(SSID string) error {
	fmt.Fprintf(os.Stdin, "password for '%s' (leave blank if open):", SSID)
	pwd, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stdin, "")
	if err != nil {
		return err
	}
	ss, err := nm.NewSettings()
	if err != nil {
		return err
	}
	cnn, err := ss.AddConnection(
		newConnectionSettings(SSID, string(pwd)))
	if err != nil {
		return err
	}
	return a.activateKnownAccessPoint(cnn, SSID)
}

func (a *WifiAdapter) activateKnownAccessPoint(
	c nm.Connection, SSID string,
) error {
	ap, err := a.accessPoint(SSID)
	if err != nil {
		return err
	}
	m, err := a.env.nm()
	if err != nil {
		return err
	}
	_, err = m.ActivateWirelessConnection(c, a.dev, ap)
	return err
}

var ErrAdapterConfigDel = errors.New(
	"wifi-adapter: delete access-point configuration")

func (a *WifiAdapter) Delete(SSID string) error {
	cnn, err := a.settingsConnectionOf(SSID)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrAdapterConfigDel, err)
	}
	if cnn == nil {
		return fmt.Errorf("%w: no configuration for '%s'",
			ErrAdapterConfigDel, SSID)
	}
	return cnn.Delete()
}

var ErrGetAccessPoint = errors.New("wifi-adapter: get access point")

func (a *WifiAdapter) accessPoint(SSID string) (nm.AccessPoint, error) {
	aa, err := a.dev.GetPropertyAccessPoints()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrGetAccessPoint, err)
	}
	if len(aa) == 0 {
		if _, err = a.Scan(); err != nil {
			return nil, fmt.Errorf("%w: %w", ErrGetAccessPoint, err)
		}
		if aa, err = a.dev.GetPropertyAccessPoints(); err != nil {
			return nil, fmt.Errorf("%w: %w", ErrGetAccessPoint, err)
		}
	}
	for _, ap := range aa {
		ssid, err := ap.GetPropertySSID()
		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrGetAccessPoint, err)
		}
		if ssid != SSID {
			continue
		}
		return ap, nil
	}
	return nil, fmt.Errorf("%w: '%s' %s", ErrGetAccessPoint, SSID,
		"not found")
}

// wirelessSettings key identifying connection settings for wifi access
// points.  NOTE NO research was done to evaluate if this is the only
// possible key for wifi connection settings.
const wirelessSettings = "802-11-wireless"

func (a *WifiAdapter) settingsConnectionOf(SSID string) (
	nm.Connection, error,
) {
	ss, err := nm.NewSettings()
	if err != nil {
		return nil, err
	}
	cc, err := ss.ListConnections()
	if err != nil {
		return nil, err
	}
	for _, c := range cc {
		ss, err := c.GetSettings()
		if err != nil {
			return nil, err
		}
		if _, ok := ss[wirelessSettings]; !ok {
			continue
		}
		ssid := string(ss[wirelessSettings]["ssid"].([]uint8))
		if ssid != SSID {
			continue
		}
		return c, nil
	}
	return nil, nil
}

var ErrAdapterPropertyChangeTimeout = errors.New(
	"property change timeout")

func (a *WifiAdapter) waitForPropertyChange(
	c chan *dbus.Signal, property string,
) error {
	for {
		select {
		case s := <-c:
			if len(s.Body) < 2 {
				continue
			}
			bodyMap, ok := s.Body[1].(map[string]dbus.Variant)
			if !ok {
				continue
			}
			_, ok = bodyMap[property]
			if !ok {
				continue
			}
			return nil
		case <-time.After(a.Timeout):
			return ErrAdapterPropertyChangeTimeout
		}
	}
}

func (a *WifiAdapter) waitForStateChange(
	c chan *dbus.Signal, state nm.NmDeviceState,
) error {
	for {
		select {
		case s := <-c:
			if len(s.Body) < 2 {
				continue
			}
			bodyMap, ok := s.Body[1].(map[string]dbus.Variant)
			if !ok {
				continue
			}
			v, ok := bodyMap["State"]
			if !ok {
				continue
			}
			st, ok := v.Value().(uint32)
			if !ok {
				continue
			}
			if nm.NmDeviceState(st) != state {
				continue
			}
			return nil
		case <-time.After(a.Timeout):
			return ErrAdapterPropertyChangeTimeout
		}
	}
}

const DBusProperties = "org.freedesktop.DBus.Properties"
const PropertiesChanged = "PropertiesChanged"

func (a *WifiAdapter) setupSignalMatcher() (
	_ chan *dbus.Signal, deferer func(e, w error) error, _ error,
) {
	cnn, err := a.lib().SystemBus()
	if err != nil {
		return nil, nil, err
	}
	err = cnn.AddMatchSignal(
		dbus.WithMatchObjectPath(a.dev.GetPath()),
		dbus.WithMatchInterface(DBusProperties),
		dbus.WithMatchMember(PropertiesChanged),
	)
	if err != nil {
		if e := cnn.Close(); e != nil {
			return nil, nil, fmt.Errorf("%w: %w", err, e)
		}
		return nil, nil, err
	}
	c := make(chan *dbus.Signal, 100)
	cnn.Signal(c)
	return c, func(err error, wrapper error) error {
		cnn.RemoveSignal(c)
		close(c)
		if e := cnn.Close(); e != nil {
			if err != nil {
				return fmt.Errorf("%w: %w", err, e)
			}
			return fmt.Errorf("%w: %w", wrapper, e)
		}
		return err

	}, nil
}

// Name returns given wifi-adapter a's name.
func (a *WifiAdapter) Name() string { return a.name }

// AdapterLib provides mockable library features.
type AdapterLib struct {
	SystemBus             func() (BusConnection, error)
	WaitForPropertyChange func(chan *dbus.Signal, string) error
	Disconnect            func() error
}

// newConnectionSettings NOTE no research was done if this basic setup
// covers all possible configuration-use-cases.
func newConnectionSettings(SSID string, pwd string) nm.ConnectionSettings {
	return nm.ConnectionSettings{
		"ipv4": map[string]interface{}{
			"method": "auto",
		},
		"ipv6": map[string]interface{}{
			"method": "auto",
		},
		"proxy": map[string]interface{}{},
		"connection": map[string]interface{}{
			"timestamp":   time.Now().UnixNano(),
			"type":        "802-11-wireless",
			"uuid":        uuid.New().String(),
			"id":          SSID,
			"permissions": []string{},
		},
		"802-11-wireless": map[string]interface{}{
			"ssid":                  []byte(SSID),
			"mac-address-blacklist": []string{},
			"mode":                  "infrastructure",
			"security":              "802-11-wireless-security",
		},
		"802-11-wireless-security": map[string]interface{}{
			"key-mgmt": "wpa-psk",
			"auth-alg": "open",
			"psk":      pwd,
		},
	}
}
