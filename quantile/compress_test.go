package quantile

import (
	"math"
	"math/rand"
	"strconv"
	"testing"
)

// compressWithSliceCopies is the pre-optimization algorithm: every merge
// removes one sample by copying the remainder of the slice.
func compressWithSliceCopies(s *stream) {
	if len(s.l) < 2 {
		return
	}
	x, xi := s.l[len(s.l)-1], len(s.l)-1
	r := s.n - 1 - x.Width
	for i := len(s.l) - 2; i >= 0; i-- {
		c := s.l[i]
		if c.Width+x.Width+x.Delta <= s.ƒ(s, r) {
			x.Width += c.Width
			s.l[xi] = x
			copy(s.l[i:], s.l[i+1:])
			s.l = s.l[:len(s.l)-1]
			xi--
		} else {
			x, xi = c, i
		}
		r -= c.Width
	}
}

func cloneStream(s *stream) *stream {
	return &stream{n: s.n, ƒ: s.ƒ, l: append(Samples(nil), s.l...)}
}

func requireSameStreamBits(t *testing.T, want, got *stream) {
	t.Helper()
	if math.Float64bits(want.n) != math.Float64bits(got.n) {
		t.Fatalf("count bits differ: want %x, got %x", math.Float64bits(want.n), math.Float64bits(got.n))
	}
	if len(want.l) != len(got.l) {
		t.Fatalf("sample count differs: want %d, got %d", len(want.l), len(got.l))
	}
	for i := range want.l {
		w, g := want.l[i], got.l[i]
		wantBits := [...]uint64{math.Float64bits(w.Value), math.Float64bits(w.Width), math.Float64bits(w.Delta)}
		gotBits := [...]uint64{math.Float64bits(g.Value), math.Float64bits(g.Width), math.Float64bits(g.Delta)}
		if wantBits != gotBits {
			t.Fatalf("sample %d bits differ: want %x, got %x", i, wantBits, gotBits)
		}
	}
}

func TestCompressMatchesSliceCopyImplementation(t *testing.T) {
	never := func(*stream, float64) float64 { return math.Inf(-1) }
	always := func(*stream, float64) float64 { return math.Inf(1) }
	rank := func(_ *stream, r float64) float64 { return r }
	notANumber := func(*stream, float64) float64 { return math.NaN() }
	targeted := NewTargeted(Targets).ƒ

	cases := []struct {
		name    string
		fn      invariant
		samples Samples
	}{
		{name: "empty", fn: targeted},
		{name: "singleton", fn: targeted, samples: Samples{{Value: math.Copysign(0, -1), Width: 1, Delta: math.Inf(1)}}},
		{name: "no-merge", fn: never, samples: Samples{
			{Value: math.Inf(-1), Width: 1},
			{Value: math.Copysign(0, -1), Width: 2, Delta: math.SmallestNonzeroFloat64},
			{Value: 0, Width: 3, Delta: math.Inf(1)},
			{Value: math.Inf(1), Width: 4},
			{Value: math.Float64frombits(0x7ff8000000000001), Width: 5},
		}},
		{name: "all-merge", fn: always, samples: Samples{{Value: -2, Width: 1}, {Value: -1, Width: 2}, {Value: 1, Width: 3}}},
		{name: "mixed", fn: rank, samples: Samples{{Value: -3, Width: 1}, {Value: -2, Width: 1}, {Value: -1, Width: 1}, {Value: 0, Width: 1}, {Value: 1, Width: 1}, {Value: 2, Width: 1}}},
		{name: "NaN-invariant", fn: notANumber, samples: Samples{{Value: 1, Width: 1}, {Value: 2, Width: 1}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			state := &stream{ƒ: tc.fn, l: append(Samples(nil), tc.samples...)}
			for _, sample := range state.l {
				state.n += sample.Width
			}
			want, got := cloneStream(state), cloneStream(state)
			compressWithSliceCopies(want)
			got.compress()
			requireSameStreamBits(t, want, got)
		})
	}

	invariants := []struct {
		name string
		fn   invariant
	}{{"targeted", targeted}, {"rank", rank}, {"never", never}, {"always", always}, {"NaN", notANumber}}
	for _, inv := range invariants {
		t.Run("random/"+inv.name, func(t *testing.T) {
			rng := rand.New(rand.NewSource(1412))
			for trial := 0; trial < 100; trial++ {
				state := &stream{ƒ: inv.fn, l: make(Samples, trial%65)}
				for i := range state.l {
					state.l[i] = Sample{Value: float64(rng.Intn(11) - 5), Width: float64(rng.Intn(4) + 1), Delta: float64(rng.Intn(8))}
					state.n += state.l[i].Width
				}
				want, got := cloneStream(state), cloneStream(state)
				compressWithSliceCopies(want)
				got.compress()
				t.Run(strconv.Itoa(trial), func(t *testing.T) { requireSameStreamBits(t, want, got) })
			}
		})
	}
}
