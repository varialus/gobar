package systemtray

import (
	"fmt"
	"time"

	"github.com/BurntSushi/xgbutil"
	"github.com/BurntSushi/xgbutil/xprop"
	"github.com/BurntSushi/xgbutil/xwindow"

	"github.com/BurntSushi/xgb/xproto"
)

// Lovingly stolen from wingo.
func currentTime(X *xgbutil.XUtil) (xproto.Timestamp, error) {
	wmClassAtom, err := xprop.Atm(X, "WM_CLASS")
	if err != nil {
		return 0, err
	}

	stringAtom, err := xprop.Atm(X, "STRING")
	if err != nil {
		return 0, err
	}

	// Make sure we're listening to PropertyChange events on the root window.
	err = xwindow.New(X, X.RootWin()).Listen(xproto.EventMaskPropertyChange)
	if err != nil {
		return 0, fmt.Errorf(
			"Could not listen to Root window events (PropertyChange): %s", err)
	}

	// Do a zero-length append on a property as suggested by ICCCM 2.1.
	err = xproto.ChangePropertyChecked(
		X.Conn(), xproto.PropModeAppend, X.RootWin(),
		wmClassAtom, stringAtom, 8, 0, nil).Check()
	if err != nil {
		return 0, err
	}

	// Now look for the PropertyNotify generated by that zero-length append
	// and return the timestamp attached to that event.
	// Note that we do this outside of xgbutil/xevent, since ownership
	// is literally the first thing we do after connecting to X.
	// (i.e., we don't have our event handling system initialized yet.)
	timeout := time.After(3 * time.Second)
	for {
		select {
		case <-timeout:
			return 0, fmt.Errorf(
				"Expected a PropertyNotify event to get a valid timestamp, " +
					"but never received one.")
		default:
			ev, err := X.Conn().PollForEvent()
			if err != nil {
				continue
			}
			if propNotify, ok := ev.(xproto.PropertyNotifyEvent); ok {
				X.TimeSet(propNotify.Time) // why not?
				return propNotify.Time, nil
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
	panic("unreachable")
}