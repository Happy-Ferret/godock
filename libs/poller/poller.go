/* Package poller is a dedicated task that handles regular polling actions.
It does not start a loop, just handles the ticker and restart channels. You
will have to get those with Start and GetRestart and use them in a loop. This
job is generaly done by the dock.StartApplet action, so you better use it or
copy and extend it to your needs.

Display and user information related to the result of the check must be made
using some return callback at the end of the check task.

Display and user information related to the check action itself, like
displaying an activity emblem during the check, should be done using the
PreCheck and PostCheck callbacks.

The goal is to keep each part separated and dedicated to one task. If we
split each role and keep it agnostic of others, we can have easier debuging
and evolution of our applets:
  * The poller send timing events.
  * The check task pull its data and send the results on a OnResult callback.
  * The OnResult callback sorts the data and dispatch it to the renderers or alert interfaces.
  * Renderer interfaces displays to the user informations or alerts the way he prefers.
*/
package poller

import "time"

//------------------------------------------------------------------[ POLLER ]--

// Poller is a dedicated task that handles regular polling actions.
//
type Poller struct {
	// Callbacks in this order.
	started   func() // Action to execute before data polling.
	callCheck func() // Action data polling.
	finished  func() // Action to execute after data polling.

	// Ticker settings.
	delay   int  // Interval between checks in second.
	enabled bool // true if the poller should be active.
	active  bool // true if the poller is really active.

	name    string      // name to send at restart
	restart chan string // restart channel to forward user requests.
}

// New creates a simple poller.
//
func New(callCheck func()) *Poller {
	poller := &Poller{
		callCheck: callCheck,
		enabled:   true,
		// restart:   make(chan bool),
	}
	return poller
}

//---------------------------------------------------------------------[ OLD ]--

// Check if polling is active.
//
//~ func (poller *Poller) Active() bool {
//~ return poller.active
//~ }

//----------------------------------------------------------------[ SETTINGS ]--

// SetPreCheck callback actions to launch before the polling job.
//
func (poller *Poller) SetPreCheck(onStarted func()) {
	poller.started = onStarted
}

// SetPostCheck callback actions to launch after the polling job.
//
func (poller *Poller) SetPostCheck(onFinished func()) {
	poller.finished = onFinished
}

// SetInterval sets the polling interval time, in seconds. You can add a default
// value as a second argument to be sure you will have a valid value (> 0).
//
func (poller *Poller) SetInterval(delay ...int) int {
	for _, d := range delay {
		if d > 0 {
			poller.delay = d
			return d
		}
	}
	poller.delay = 3600 * 24 // Failed to provide a valid value. Set check interval to a day.
	return poller.delay
}

func (poller *Poller) GetInterval() int {
	return poller.delay
}

// ChanRestart is the restart event channel. You will need to lock it with Wait
// in a select loop to have a real polling routine.
//
// func (poller *Poller) ChanRestart() chan bool {
// 	return poller.restart
// }

func (poller *Poller) SetChanRestart(c chan string, name string) {
	poller.restart = c
	poller.name = name
}

//------------------------------------------------------------------[ ACTION ]--

// Wait return a channel that will be triggered after the defined poller interval.
// You will have to call it on every loop as it not a real ticker. It's just a
// single use chan.
//
// func (poller *Poller) Wait() <-chan bool {
func (poller *Poller) Wait() <-chan time.Time {
	if poller.enabled && poller.delay > 0 {
		poller.active = true
		return time.After(time.Duration(poller.delay) * time.Second)
	}
	return nil
}

// Restart polling ticker. This will send an event on the restart channel.
//
func (poller *Poller) Restart() {
	if poller.enabled {
		// poller.Stop()
		poller.restart <- poller.name // send our restart event.
		// poller.enabled = true
		// poller.active = true
	}
}

// Stop the polling ticker.
//
func (poller *Poller) Stop() {
	if poller.active {
		poller.enabled = false
		poller.active = false
	}
}

// Check action. Launch PreCheck, OnCheck and PostCheck callbacks.
//
func (poller *Poller) Action() {
	if poller.started != nil { // Pre check call.
		poller.started()
	}

	poller.callCheck() // Data check call. Does the real polling job.

	if poller.finished != nil { // Post check call.
		poller.finished()
	}
}
