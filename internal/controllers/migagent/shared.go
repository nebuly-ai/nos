package migagent

import "sync"

type empty struct{}

// SharedState contains the information shared between the Actuator and the Reporter processes
type SharedState struct {
	sync.Mutex
	reportsChan chan empty
}

func NewSharedState() *SharedState {
	return &SharedState{
		reportsChan: make(chan empty, 1),
	}
}

func (s *SharedState) OnReportDone() {
	select {
	case s.reportsChan <- struct{}{}:
	default:
	}
}

func (s *SharedState) OnApplyDone() {
	select {
	case <-s.reportsChan:
	default:
	}
}

func (s *SharedState) AtLeastOneReportSinceLastApply() bool {
	select {
	case <-s.reportsChan: // it means that at least one report has been performed since the last Apply
		return true
	default:
		return false
	}
}
