package main

import "time"

// One bounded record per configured target. Group results belong to the actual
// group request, not to unrelated later commands to its members.
type controlRequest struct {
	id      uint64
	action  string
	at      int64
	members map[string]string
}

type GroupState struct {
	DeviceState
	Total     int `json:"total"`
	Sent      int `json:"sent"`
	Failed    int `json:"failed"`
	Cancelled int `json:"cancelled"`
	Pending   int `json:"pending"`
}

// All helpers run with the server mutex held.
func (s *Server) beginRequest(target, action string, ids []string) uint64 {
	if s.requests == nil {
		s.requests = make(map[string]*controlRequest)
		s.latestDevice = make(map[string]uint64)
	}
	s.nextRequest++
	r := &controlRequest{id: s.nextRequest, action: action, at: time.Now().Unix(), members: make(map[string]string)}
	for _, id := range ids {
		r.members[id] = "Queued"
	}
	s.requests[target] = r
	return r.id
}

func (s *Server) finishCommand(c Command, result string, now time.Time) {
	if s.latestDevice[c.Device] == c.Request {
		s.states[c.Device] = DeviceState{c.Action, result, now.Unix()}
	}
	if r := s.requests[c.Target]; r != nil && r.id == c.Request {
		r.members[c.Device] = result
		r.at = now.Unix()
	}
}

func (s *Server) groupStates() map[string]GroupState {
	out := make(map[string]GroupState)
	for _, g := range s.config.Groups {
		r := s.requests[g.ID]
		if r == nil {
			continue
		}
		state := GroupState{DeviceState: DeviceState{r.action, "Complete", r.at}, Total: len(r.members)}
		for _, result := range r.members {
			switch result {
			case "Queued":
				state.Pending++
			case "Sent (position unknown)":
				state.Sent++
			case "Superseded", "Schedule cancelled":
				state.Cancelled++
			default:
				state.Failed++
			}
		}
		if state.Pending > 0 {
			state.Status = "Queued"
		}
		out[g.ID] = state
	}
	return out
}
