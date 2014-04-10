package timeout

import "time"

type TimeoutMessage int

const (
	TimeoutResetMessage TimeoutMessage = iota
	TimeoutTriggerMessage
	TimeoutCancelMessage
)

type Timeout struct {
	timeoutChan chan bool
}

func NewTimeout(duration time.Duration, timeoutFunc func()) *Timeout {
	timeout := &Timeout{}

	timeout.resetTimeoutChan()

	go func() {
		for {
			switch <-timeout.timeoutChan {
			case TimeoutResetMessage:
				// make a new chan so that the old TimeoutTriggerMessage won't reach us
				timeout.resetTimeoutChan()

				go func() {
					time.Sleep(self.duration)

					timeout.timeoutChan <- TimeoutTriggerMessage
				}()
			case TimeoutTriggerMessage:
				timeoutFunc()
				break
			case TimeoutCancelMessage:
				break
			}
		}
	}()

	timeout.Reset()

	return timeout
}

func (self *Timeout) resetTimeoutChan() {
	self.timeoutChan = make(chan TimeoutMessage, 1)
}

func (self *Timeout) Reset() {
	self.timeoutChan <- TimeoutResetMessage
}

func (self *Timeout) Cancel() {
	self.timeoutChan <- TimeoutCancelMessage
}
