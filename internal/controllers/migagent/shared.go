/*
 * Copyright 2022 Nebuly.ai
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package migagent

import "sync"

type empty struct{}

// SharedState contains the information shared between the Actuator and the Reporter processes
type SharedState struct {
	sync.Mutex
	lastParsedPlanId string
	reportsChan      chan empty
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
