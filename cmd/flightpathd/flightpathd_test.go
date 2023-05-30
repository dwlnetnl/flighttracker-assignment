package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/dwlnetnl/flighttracker-assignment/flightpath"
)

var unmarshalTests = []struct {
	in   string
	want []*flightpath.Flight
}{
	{`[["SFO", "EWR"]]`,
		[]*flightpath.Flight{
			{From: "SFO", To: "EWR"},
		}},
	{`[["ATL", "EWR"], ["SFO", "ATL"]]`,
		[]*flightpath.Flight{
			{From: "ATL", To: "EWR"},
			{From: "SFO", To: "ATL"},
		}},
	{`[["IND", "EWR"], ["SFO", "ATL"], ["GSO", "IND"], ["ATL", "GSO"]]`,
		[]*flightpath.Flight{
			{From: "IND", To: "EWR"},
			{From: "SFO", To: "ATL"},
			{From: "GSO", To: "IND"},
			{From: "ATL", To: "GSO"},
		}},
}

var unmarshalMethods = []struct {
	name string
	fn   func([]byte) ([]*flightpath.Flight, error)
}{
	{"json", unmarshalJSON},
	{"regexp", unmarshalRegexp},
	{"custom", unmarshalCustom},
}

func TestUnmarshal(t *testing.T) {
	for _, m := range unmarshalMethods {
		t.Run(m.name, func(t *testing.T) {
			for _, c := range unmarshalTests {
				t.Run(c.in, func(t *testing.T) {
					got, err := m.fn([]byte(c.in))
					if err != nil {
						t.Fatal(err)
					}
					if diff := cmp.Diff(c.want, got); diff != "" {
						t.Errorf("mismatch (-want +got):\n%s", diff)
					}
				})
			}
		})
	}
}

func BenchmarkUnmarshal(b *testing.B) {
	for _, m := range unmarshalMethods {
		b.Run(m.name, func(b *testing.B) {
			for _, c := range unmarshalTests {
				in := []byte(c.in)
				b.Run(c.in, func(b *testing.B) {
					b.ReportAllocs()
					b.ResetTimer()
					for i := 0; i < b.N; i++ {
						_, _ = m.fn(in)
					}
				})
			}
		})
	}
}

const marshalResult = `["SFO","EWR"]` + "\n"

var marshalMethods = []struct {
	name string
	fn   func(w io.Writer, f *flightpath.Flight) error
}{
	{"json", marshalJSON},
	{"fmt", marshalFmt},
	{"append", marshalAppend},
}

func TestMarshal(t *testing.T) {
	in := &flightpath.Flight{From: "SFO", To: "EWR"}
	var b strings.Builder
	for _, m := range marshalMethods {
		t.Run(m.name, func(t *testing.T) {
			b.Reset()

			err := m.fn(&b, in)
			if err != nil {
				t.Fatal(err)
			}

			got := b.String()
			const want = marshalResult
			if got != want {
				t.Errorf("\ngot:  %q\nwant: %q\n",
					strings.TrimSpace(got), strings.TrimSpace(want))
			}
		})
	}
}

func BenchmarkMarshal(b *testing.B) {
	f := &flightpath.Flight{From: "SFO", To: "EWR"}
	for _, m := range marshalMethods {
		b.Run(m.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = m.fn(io.Discard, f)
			}
		})
	}
}

func testResponse(t *testing.T, r *http.Response) {
	t.Helper()

	if got, want := r.StatusCode, http.StatusOK; got != want {
		t.Errorf("got %d, want: %d", got, want)
	}

	data, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatal(err)
	}

	got := string(data)
	const want = marshalResult
	if got != want {
		t.Errorf("\ngot:  %q\nwant: %q\n",
			strings.TrimSpace(got), strings.TrimSpace(want))
	}
}

func TestCalculate(t *testing.T) {
	for _, c := range unmarshalTests {
		t.Run(c.in, func(t *testing.T) {
			body := strings.NewReader(c.in)
			req := httptest.NewRequest(http.MethodPost, "/calculate", body)
			rec := httptest.NewRecorder()
			calculate(rec, req)

			if got, want := rec.Code, http.StatusOK; got != want {
				t.Errorf("got %d, want: %d", got, want)
			}

			testResponse(t, rec.Result())
		})
	}
}

func TestServer(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(calculate))
	t.Cleanup(ts.Close)

	for _, c := range unmarshalTests {
		t.Run(c.in, func(t *testing.T) {
			body := strings.NewReader(c.in)
			res, err := http.Post(ts.URL, "application/json", body)
			if err != nil {
				t.Fatal(err)
			}

			testResponse(t, res)
		})
	}
}
