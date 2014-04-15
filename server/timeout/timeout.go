package timeout

import "time"

type responseChanType chan bool
type controlChanType chan responseChanType

type Timeout struct {
	resetChan  controlChanType
	cancelChan controlChanType
}

func NewTimeout(duration time.Duration, timeoutFunc func()) *Timeout {
	timeout := &Timeout{
		resetChan:  make(controlChanType),
		cancelChan: make(controlChanType),
	}

	go func() {
		for {
			select {
			case <-time.After(duration):
				timeoutFunc()
				return
			case responseChan := <-timeout.resetChan:
				responseChan <- true
			case responseChan := <-timeout.cancelChan:
				responseChan <- true
				return
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

func (self *Timeout) Reset() bool {
	responseChan := make(responseChanType)

	self.resetChan <- responseChan

	return <-responseChan
}

func (self *Timeout) Cancel() bool {
	responseChan := make(responseChanType)

	self.cancelChan <- responseChan

	return <-responseChan
}
