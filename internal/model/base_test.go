package model

import (
	"database/sql/driver"
	"encoding/json"
	"testing"
	"time"
)

func TestUpsertTime_MarshalJSON(t *testing.T) {
	tests := []struct {
		name string
		time UpsertTime
		want string
	}{
		{
			name: "normal time",
			time: UpsertTime(time.Date(2024, 6, 15, 10, 30, 0, 0, time.Local)),
			want: `"2024-06-15 10:30:00"`,
		},
		{
			name: "zero time",
			time: UpsertTime{},
			want: `"0001-01-01 00:00:00"`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.time.MarshalJSON()
			if err != nil {
				t.Fatalf("MarshalJSON() error = %v", err)
			}
			if string(got) != tt.want {
				t.Errorf("MarshalJSON() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestUpsertTime_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    time.Time
		wantErr bool
	}{
		{
			name:  "normal time",
			input: `"2024-06-15 10:30:00"`,
			want:  time.Date(2024, 6, 15, 10, 30, 0, 0, time.Local),
		},
		{
			name:  "null",
			input: `null`,
			want:  time.Time{},
		},
		{
			name:  "empty string",
			input: `""`,
			want:  time.Time{},
		},
		{
			name:    "invalid format",
			input:   `"not-a-time"`,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got UpsertTime
			err := got.UnmarshalJSON([]byte(tt.input))
			if (err != nil) != tt.wantErr {
				t.Fatalf("UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && !time.Time(got).Equal(tt.want) {
				t.Errorf("UnmarshalJSON() = %v, want %v", time.Time(got), tt.want)
			}
		})
	}
}

func TestUpsertTime_JSONRoundTrip(t *testing.T) {
	type wrapper struct {
		T UpsertTime `json:"t"`
	}
	original := wrapper{T: UpsertTime(time.Date(2025, 1, 2, 15, 4, 5, 0, time.Local))}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var decoded wrapper
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal error: %v", err)
	}

	if !time.Time(decoded.T).Equal(time.Time(original.T)) {
		t.Errorf("round trip failed: got %v, want %v", time.Time(decoded.T), time.Time(original.T))
	}
}

func TestUpsertTime_Value(t *testing.T) {
	tests := []struct {
		name string
		time UpsertTime
		want driver.Value
	}{
		{
			name: "normal time",
			time: UpsertTime(time.Date(2024, 6, 15, 10, 30, 0, 0, time.Local)),
			want: time.Date(2024, 6, 15, 10, 30, 0, 0, time.Local),
		},
		{
			name: "zero time returns nil",
			time: UpsertTime{},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.time.Value()
			if err != nil {
				t.Fatalf("Value() error = %v", err)
			}
			if tt.want == nil {
				if got != nil {
					t.Errorf("Value() = %v, want nil", got)
				}
			} else {
				gotTime, ok := got.(time.Time)
				if !ok {
					t.Fatalf("Value() returned %T, want time.Time", got)
				}
				if !gotTime.Equal(tt.want.(time.Time)) {
					t.Errorf("Value() = %v, want %v", gotTime, tt.want)
				}
			}
		})
	}
}

func TestUpsertTime_Scan(t *testing.T) {
	refTime := time.Date(2024, 6, 15, 10, 30, 0, 0, time.Local)

	tests := []struct {
		name    string
		input   interface{}
		want    time.Time
		wantErr bool
	}{
		{
			name:  "nil",
			input: nil,
			want:  time.Time{},
		},
		{
			name:  "time.Time",
			input: refTime,
			want:  refTime,
		},
		{
			name:  "*time.Time",
			input: &refTime,
			want:  refTime,
		},
		{
			name:  "*time.Time nil",
			input: (*time.Time)(nil),
			want:  time.Time{},
		},
		{
			name:  "[]byte valid",
			input: []byte("2024-06-15 10:30:00"),
			want:  refTime,
		},
		{
			name:  "[]byte empty",
			input: []byte(""),
			want:  time.Time{},
		},
		{
			name:    "[]byte invalid",
			input:   []byte("invalid"),
			wantErr: true,
		},
		{
			name:  "string valid",
			input: "2024-06-15 10:30:00",
			want:  refTime,
		},
		{
			name:  "string empty",
			input: "",
			want:  time.Time{},
		},
		{
			name:    "string invalid",
			input:   "invalid",
			wantErr: true,
		},
		{
			name:    "unsupported type",
			input:   123,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got UpsertTime
			err := got.Scan(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Scan() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && !time.Time(got).Equal(tt.want) {
				t.Errorf("Scan() = %v, want %v", time.Time(got), tt.want)
			}
		})
	}
}
