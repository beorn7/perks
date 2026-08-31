package quantile

import "testing"

func BenchmarkInsertTargeted(b *testing.B) {
	b.ReportAllocs()

	s := NewTargeted(Targets)
	b.ResetTimer()
	for i := float64(0); i < float64(b.N); i++ {
		s.Insert(i)
	}
}

func BenchmarkInsertTargetedSmallEpsilon(b *testing.B) {
	s := NewTargeted(TargetsSmallEpsilon)
	b.ResetTimer()
	for i := float64(0); i < float64(b.N); i++ {
		s.Insert(i)
	}
}

func BenchmarkInsertBiased(b *testing.B) {
	s := NewLowBiased(0.01)
	b.ResetTimer()
	for i := float64(0); i < float64(b.N); i++ {
		s.Insert(i)
	}
}

func BenchmarkInsertBiasedSmallEpsilon(b *testing.B) {
	s := NewLowBiased(0.0001)
	b.ResetTimer()
	for i := float64(0); i < float64(b.N); i++ {
		s.Insert(i)
	}
}

func BenchmarkQuery(b *testing.B) {
	s := NewTargeted(Targets)
	for i := float64(0); i < 1e6; i++ {
		s.Insert(i)
	}
	b.ResetTimer()
	n := float64(b.N)
	for i := float64(0); i < n; i++ {
		s.Query(i / n)
	}
}

func BenchmarkQuerySmallEpsilon(b *testing.B) {
	s := NewTargeted(TargetsSmallEpsilon)
	for i := float64(0); i < 1e6; i++ {
		s.Insert(i)
	}
	b.ResetTimer()
	n := float64(b.N)
	for i := float64(0); i < n; i++ {
		s.Query(i / n)
	}
}

// BenchmarkInsertTargetedSparseOutliers reproduces the long equal-value runs that
// previously caused repeated tail copies during compression.
func BenchmarkInsertTargetedSparseOutliers(b *testing.B) {
	const inserts = 500000
	for _, tc := range []struct {
		name  string
		value func(int) float64
	}{
		{name: "all-zero-control", value: func(int) float64 { return 0 }},
		{name: "sparse-outliers", value: func(i int) float64 {
			if i%500 == 0 {
				return 2
			}
			return 0
		}},
	} {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			for iteration := 0; iteration < b.N; iteration++ {
				s := NewTargeted(Targets)
				for i := 0; i < inserts; i++ {
					s.Insert(tc.value(i))
				}
			}
		})
	}
}
