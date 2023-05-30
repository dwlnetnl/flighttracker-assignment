package flightpath

import (
	"fmt"
	"testing"
)

var calculationTests = []struct {
	in   []*Flight
	want *Flight
}{
	{[]*Flight{
		{"SFO", "EWR"},
	}, &Flight{"SFO", "EWR"}},

	{[]*Flight{
		{"ATL", "EWR"},
		{"SFO", "ATL"},
	}, &Flight{"SFO", "EWR"}},

	{[]*Flight{
		{"IND", "EWR"},
		{"SFO", "ATL"},
		{"GSO", "IND"},
		{"ATL", "GSO"},
	}, &Flight{"SFO", "EWR"}},

	// added for calculateReduce
	{[]*Flight{
		{"SFO", "ATL"},
		{"ATL", "EWR"},
	}, &Flight{"SFO", "EWR"}},
	{[]*Flight{
		{"SFO", "ATL"},
		{"IND", "EWR"},
		{"GSO", "IND"},
		{"ATL", "GSO"},
	}, &Flight{"SFO", "EWR"}},
}

func Test_calculateSort(t *testing.T) {
	for _, c := range calculationTests {
		name := fmt.Sprintf("%v", c.in)
		t.Run(name, func(t *testing.T) {
			// copy input to guard against changing test data
			in := make([]*Flight, len(c.in))
			copy(in, c.in)

			got := calculateSort(in)
			if *got != *c.want {
				t.Errorf("got %v, want: %v", got, c.want)
			}
		})
	}
}

func Test_calculateReduce(t *testing.T) {
	for _, c := range calculationTests {
		name := fmt.Sprintf("%v", c.in)
		t.Run(name, func(t *testing.T) {
			got := calculateReduce(c.in)
			if *got != *c.want {
				t.Errorf("got %v, want: %v", got, c.want)
			}
		})
	}
}

func BenchmarkCalculate(b *testing.B) {
	b.Run("sort", func(b *testing.B) {
		for _, c := range calculationTests {
			name := fmt.Sprintf("%v", c.in)
			b.Run(name, func(b *testing.B) {
				// copy input to guard against changing test data
				in := make([][]*Flight, 1024)
				for i := 0; i < len(in); i++ {
					in[i] = make([]*Flight, len(c.in))
				}

				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					// copy input to guard against changing test data
					idx := i % len(in)
					if idx == 0 {
						b.StopTimer()
						for i := 0; i < len(in); i++ {
							copy(in[i], c.in)
						}
						b.StartTimer()
					}

					_ = calculateSort(in[idx])
				}
			})
		}
	})
	b.Run("reduce", func(b *testing.B) {
		for _, c := range calculationTests {
			name := fmt.Sprintf("%v", c.in)
			b.Run(name, func(b *testing.B) {
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					_ = calculateReduce(c.in)
				}
			})
		}
	})
}
