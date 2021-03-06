package timeout

import "time"

// Timeout is an interface for objects that facilitate managing active timeouts, allowing resetting and canceling
type Timeout interface {
	Reset() bool
	Cancel() bool
}

type responseChanType chan bool
type controlChanType chan responseChanType

type timeout struct {
	resetChan  controlChanType
	cancelChan controlChanType
}

// New creates a Timeout, which calls timeoutFunc after duration.
// It can be reset or cancelled.
func New(duration time.Duration, timeoutFunc func()) Timeout {
	timeout := &timeout{
		resetChan:  make(controlChanType),
		cancelChan: make(controlChanType),
	}

	go func() {
	ActiveLoop:
		for {
			select {
			case <-time.After(duration):
				timeoutFunc()
				break ActiveLoop
			case responseChan := <-timeout.resetChan:
				responseChan <- true
			case responseChan := <-timeout.cancelChan:
				responseChan <- true
				break ActiveLoop
			}
		}

		// now that the timeout has been triggered or cancelled, Reset and Cancel will be returning false
		for {
			select {
			case responseChan := <-timeout.resetChan:
				responseChan <- false
			case responseChan := <-timeout.cancelChan:
				responseChan <- false
			}
		}
	}()

	return timeout
}

func (t *timeout) Reset() bool {
	responseChan := make(responseChanType)

	t.resetChan <- responseChan

	return <-responseChan
}

func (t *timeout) Cancel() bool {
	responseChan := make(responseChanType)

	t.cancelChan <- responseChan

	return <-responseChan
}
