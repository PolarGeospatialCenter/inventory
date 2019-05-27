package types

import (
	"testing"
	"time"
)

func TestIPReservationValidAt(t *testing.T) {

	startTimeLocal := time.Date(2019, 05, 23, 05, 23, 34, 0, time.Local)
	endTimeLocal := time.Date(2019, 05, 23, 06, 0, 34, 0, time.Local)
	cases := []struct {
		name  string
		r     IPReservation
		t     time.Time
		valid bool
	}{
		{
			name:  "Invalid Before Start",
			r:     IPReservation{Start: &startTimeLocal, End: nil},
			t:     time.Date(2019, 05, 23, 05, 23, 33, 0, time.Local),
			valid: false,
		},
		{
			name:  "Valid After Start",
			r:     IPReservation{Start: &startTimeLocal, End: nil},
			t:     time.Date(2019, 05, 23, 05, 23, 35, 0, time.Local),
			valid: true,
		},
		{
			name:  "Valid After Start and before End",
			r:     IPReservation{Start: &startTimeLocal, End: &endTimeLocal},
			t:     time.Date(2019, 05, 23, 05, 23, 35, 0, time.Local),
			valid: true,
		},
		{
			name:  "Invalid After End",
			r:     IPReservation{Start: &startTimeLocal, End: &endTimeLocal},
			t:     time.Date(2019, 05, 23, 06, 1, 35, 0, time.Local),
			valid: false,
		},
		{
			name:  "Invalid Before Start",
			r:     IPReservation{Start: &startTimeLocal, End: &endTimeLocal},
			t:     time.Date(2019, 05, 23, 05, 22, 35, 0, time.Local),
			valid: false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(st *testing.T) {
			valid := c.r.ValidAt(c.t)
			if valid != c.valid {
				st.Errorf("Expected ValidAt to return %v, actually returned %v", c.valid, valid)
			}
		})
	}
}
