package api

import (
	"bytes"
	"encoding/json"
	"testing"

	"bookmark-cli/internal/cli"
)

// Keep API date strings intact through decoding and the CLI's JSON output.
// In particular, +00:00 must not be normalized to Z by a time.Time round trip.
func TestResponseDateTimesPreserved(t *testing.T) {
	for _, offset := range []string{"+09:00", "+00:00", "Z", "-04:00"} {
		for _, tc := range []struct {
			name   string
			new    func() any
			fields []string
		}{
			{"bookmark", func() any { return &Bookmark{} }, []string{"created_at", "updated_at"}},
			{"user", func() any { return &User{} }, []string{"created_at", "updated_at"}},
			{"session", func() any { return &SessionDetail{} }, []string{"created_at", "last_used_at", "expires_at"}},
		} {
			t.Run(tc.name+"/"+offset, func(t *testing.T) {
				want := make(map[string]string)
				for i, field := range tc.fields {
					// Distinct timestamps also detect swapped fields.
					want[field] = []string{"2026-10-01T12:34:56", "2026-10-02T09:45:01", "2026-10-08T00:00:00"}[i] + offset
				}
				response, err := json.Marshal(want)
				if err != nil {
					t.Fatal(err)
				}
				model := tc.new()
				if err := json.Unmarshal(response, model); err != nil {
					t.Fatal(err)
				}
				var output bytes.Buffer
				if err := cli.PrintJSONTo(&output, model); err != nil {
					t.Fatal(err)
				}
				var got map[string]json.RawMessage
				if err := json.Unmarshal(output.Bytes(), &got); err != nil {
					t.Fatal(err)
				}
				for field, date := range want {
					var actual string
					if err := json.Unmarshal(got[field], &actual); err != nil {
						t.Fatal(err)
					}
					if actual != date {
						t.Errorf("%s = %q, want %q", field, actual, date)
					}
				}
			})
		}
	}
}
