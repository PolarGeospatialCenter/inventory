package types

import (
	"encoding/json"
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

func TestUnmarshalIPReservationJSON(t *testing.T) {
	jsonString := `{
		"HostInformation": "",
		"start": "2019-06-03T19:27:02Z",
		"end": null,
		"metadata": {
			"note": "metadata field",
			"number": 5,
			"bool": true,
			"null": null
		},
		"ip": "10.0.0.1/27",
		"mac": "",
		"gateway": "10.0.0.30",
		"dns": [
			"1.1.1.1",
			"8.8.8.8"
		]
	}`

	reservation := &IPReservation{}

	err := json.Unmarshal([]byte(jsonString), reservation)
	if err != nil {
		t.Errorf("failed to unmarshal ip reservation: %v", err)
	}
}
