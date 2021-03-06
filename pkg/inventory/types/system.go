package types

import "time"

type System struct {
	Name         string
	ShortName    string
	Environments map[string]*Environment
	Roles        []string
	Metadata     Metadata
	LastUpdated  time.Time
}

func NewSystem() *System {
	return &System{}
}

func (s *System) ID() string {
	if s.ShortName != "" {
		return s.ShortName
	} else {
		return s.Name
	}
}

func (s *System) Timestamp() int64 {
	return s.LastUpdated.Unix()
}

func (s *System) SetTimestamp(timestamp time.Time) {
	s.LastUpdated = timestamp
}
