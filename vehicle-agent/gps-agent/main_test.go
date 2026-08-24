package main

import (
	"math"
	"testing"
)

func TestNMEACoordinate(t *testing.T) {
	tests := []struct {
		raw, hemisphere string
		latitude        bool
		want            float64
	}{
		{"2409.2689", "N", true, 24.1544816667},
		{"12038.8827", "E", false, 120.648045},
		{"2409.2689", "S", true, -24.1544816667},
		{"12038.8827", "W", false, -120.648045},
	}
	for _, test := range tests {
		got, err := nmeaCoordinate(test.raw, test.hemisphere, test.latitude)
		if err != nil {
			t.Fatalf("convert %s%s: %v", test.raw, test.hemisphere, err)
		}
		if math.Abs(got-test.want) > 0.0000001 {
			t.Fatalf("convert %s%s: got %.10f, want %.10f", test.raw, test.hemisphere, got, test.want)
		}
	}
}

func TestParseGGAWithRMCState(t *testing.T) {
	state := nmeaState{}
	updateRMC(&state, nmeaFields("$GPRMC,095404.00,A,2409.2689,N,12038.8827,E,1.5,166.08,030826,,,A*00"))
	payload, ok := parseGGA(nmeaFields("$GPGGA,095404.00,2409.2689,N,12038.8827,E,1,03,1.7,112.5,M,16.0,M,,*00"), state)
	if !ok {
		t.Fatal("expected a valid GGA fix")
	}
	if payload.FixQuality != 1 || payload.Satellites == nil || *payload.Satellites != 3 {
		t.Fatalf("unexpected fix metadata: %+v", payload)
	}
	if math.Abs(payload.SpeedKmh-2.778) > 0.0001 {
		t.Fatalf("unexpected speed: %f", payload.SpeedKmh)
	}
	if payload.RecordedAt == nil || payload.RecordedAt.Year() != 2026 {
		t.Fatalf("unexpected recorded time: %v", payload.RecordedAt)
	}
}

func TestParseGGARejectsNoFix(t *testing.T) {
	_, ok := parseGGA(nmeaFields("$GPGGA,,,,,,0,,,,,,,,*00"), nmeaState{})
	if ok {
		t.Fatal("expected no-fix GGA to be rejected")
	}
}
