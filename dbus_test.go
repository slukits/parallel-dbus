/* helper for dbus debugging */

package main

import (
	"fmt"
	"strings"
	"sync"

	"github.com/godbus/dbus/v5"
)

type DBusSignalLogger struct {
	*sync.Mutex
	log           []string
	signalChannel chan *dbus.Signal
	conn          *dbus.Conn
}

// newDBusSignalLogger creates/returns a new DBusSignalLogger-instance
// and starts logging asynchronously.  interface_ and signal may be zero
// in which case they default to 'org.freedesktop.DBus.Properties' and
// 'PropertiesChanged' respectively.
func newDBusSignalLogger(
	path dbus.ObjectPath, interface_, signal string,
) (*DBusSignalLogger, error) {
	if interface_ == "" {
		interface_ = "org.freedesktop.DBus.Properties"
	}
	if signal == "" {
		signal = "PropertiesChanged"
	}
	conn, err := dbus.ConnectSystemBus()
	if err != nil {
		return nil, err
	}
	err = conn.AddMatchSignal(
		dbus.WithMatchObjectPath(path),
		dbus.WithMatchInterface(interface_),
		dbus.WithMatchMember(signal),
	)
	if err != nil {
		return nil, err
	}
	c := make(chan *dbus.Signal, 100)
	conn.Signal(c)
	sl := &DBusSignalLogger{
		Mutex:         &sync.Mutex{},
		signalChannel: c,
		conn:          conn,
	}
	go sl.logger()
	return sl, nil
}

// Log returns so far logged signals
func (l *DBusSignalLogger) Log() string {
	l.Lock()
	defer l.Unlock()
	return strings.Join(l.log, "\n")
}

// logger should be started in its own go-routine
func (l *DBusSignalLogger) logger() {
	for s := range l.signalChannel {
		if s == nil {
			return
		}
		l.Lock()
		if len(s.Body) < 2 {
			l.Unlock()
			continue
		}
		bodyMap, ok := s.Body[1].(map[string]dbus.Variant)
		if !ok {
			l.Unlock()
			continue
		}
		l.log = append(l.log, fmt.Sprintf("%#v", s.Body[0]))
		for k, v := range bodyMap {
			l.log = append(l.log, fmt.Sprintf(
				"%s: %T: %s", k, v.Value(), v.String()))
		}
		l.log = append(l.log, "")
		l.Unlock()
	}
}

// Clear removes all loggings to the point it is called and returns the
// removed loggings.
func (l *DBusSignalLogger) Clear() string {
	l.Lock()
	defer l.Unlock()
	log := strings.Join(l.log, "\n")
	l.log = []string{}
	return log
}

// Close stops logging, closes the system dbus connection and returns
// logged signals.
func (l *DBusSignalLogger) Close() (string, error) {
	l.Lock()
	defer l.Unlock()
	log := strings.Join(l.log, "\n")
	l.conn.RemoveSignal(l.signalChannel)
	err := l.conn.Close()
	return log, err
}
