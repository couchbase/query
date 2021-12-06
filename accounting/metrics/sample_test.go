package metrics

import (
	"math/rand"
	"runtime"
	"testing"
	"time"
)

func BenchmarkExpDecaySample257(b *testing.B) {
	benchmarkSample(b, NewExpDecaySample(257, 0.015))
}

func BenchmarkExpDecaySample514(b *testing.B) {
	benchmarkSample(b, NewExpDecaySample(514, 0.015))
}

func BenchmarkExpDecaySample1028(b *testing.B) {
	benchmarkSample(b, NewExpDecaySample(1028, 0.015))
}

func TestExpDecaySample10(t *testing.T) {
	s := NewExpDecaySample(104, 0.99)
	setSeed(s)
	for i := 0; i < 10; i++ {
		s.Update(int64(i))
	}
	if size := s.Count(); 10 != size {
		t.Errorf("s.Count(): 10 != %v\n", size)
	}
	if size := s.Size(); 10 != size {
		t.Errorf("s.Size(): 10 != %v\n", size)
	}
	if l := len(s.Values()); 10 != l {
		t.Errorf("len(s.Values()): 10 != %v\n", l)
	}
	for _, v := range s.Values() {
		if v > 10 || v < 0 {
			t.Errorf("out of range [0, 10): %v\n", v)
		}
	}
}

func TestExpDecaySample100(t *testing.T) {
	s := NewExpDecaySample(1000, 0.01)
	setSeed(s)
	for i := 0; i < 100; i++ {
		s.Update(int64(i))
	}
	if size := s.Count(); 100 != size {
		t.Errorf("s.Count(): 100 != %v\n", size)
	}
	if size := s.Size(); 100 != size {
		t.Errorf("s.Size(): 100 != %v\n", size)
	}
	if l := len(s.Values()); 100 != l {
		t.Errorf("len(s.Values()): 100 != %v\n", l)
	}
	for _, v := range s.Values() {
		if v > 100 || v < 0 {
			t.Errorf("out of range [0, 100): %v\n", v)
		}
	}
}

func TestExpDecaySample1000(t *testing.T) {
	s := NewExpDecaySample(104, 0.99)
	setSeed(s)
	for i := 0; i < 1000; i++ {
		s.Update(int64(i))
	}
	if size := s.Count(); 1000 != size {
		t.Errorf("s.Count(): 1000 != %v\n", size)
	}
	if size := s.Size(); 104 != size {
		t.Errorf("s.Size(): 104 != %v\n", size)
	}
	if l := len(s.Values()); 104 != l {
		t.Errorf("len(s.Values()): 104 != %v\n", l)
	}
	for _, v := range s.Values() {
		if v > 1000 || v < 0 {
			t.Errorf("out of range [0, 1000): %v\n", v)
		}
	}
}

// This test makes sure that the sample's priority is not amplified by using
// nanosecond duration since start rather than second duration since start.
// The priority becomes +Inf quickly after starting if this is done,
// effectively freezing the set of samples until a rescale step happens.
func TestExpDecaySampleNanosecondRegression(t *testing.T) {
	s := NewExpDecaySample(100, 0.99)
	setSeed(s)
	for i := 0; i < 100; i++ {
		s.Update(10)
	}
	time.Sleep(1 * time.Millisecond)
	for i := 0; i < 100; i++ {
		s.Update(20)
	}
	v := s.Values()
	avg := float64(0)
	for i := 0; i < len(v); i++ {
		avg += float64(v[i])
	}
	avg /= float64(len(v))
	if avg > 16 || avg < 14 {
		t.Errorf("out of range [14, 16]: %v\n", avg)
	}
}

func TestExpDecaySampleRescale(t *testing.T) {
	s := NewExpDecaySample(2, 0.001).(*ExpDecaySample)
	s.UpdateWithTimestamp(time.Now(), 1)
	s.UpdateWithTimestamp(time.Now().Add(time.Hour+time.Microsecond), 1)
	s.scan(func(v expDecaySample) {
		if v.k == 0.0 {
			t.Fatal("v.k == 0.0")
		}
	})
}

func TestExpDecaySampleStatistics(t *testing.T) {
	now := time.Now()
	s := NewExpDecaySample(100, 0.99)
	setSeed(s)
	for i := 1; i <= 10000; i++ {
		s.UpdateWithTimestamp(now.Add(time.Duration(i)), int64(i))
	}
	testExpDecaySampleStatistics(t, s)
}

func benchmarkSample(b *testing.B, s Sample) {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	pauseTotalNs := memStats.PauseTotalNs
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Update(1)
	}
	b.StopTimer()
	runtime.GC()
	runtime.ReadMemStats(&memStats)
	b.Logf("GC cost: %d ns/op", int(memStats.PauseTotalNs-pauseTotalNs)/b.N)
}

func testExpDecaySampleStatistics(t *testing.T, s Sample) {
	if count := s.Count(); 10000 != count {
		t.Errorf("s.Count(): 10000 != %v\n", count)
	}
	if min := s.Min(); 81 != min {
		t.Errorf("s.Min(): 81 != %v\n", min)
	}
	if max := s.Max(); 10000 != max {
		t.Errorf("s.Max(): 10000 != %v\n", max)
	}
	if mean := s.Mean(); 5151.807692307692 != mean {
		t.Errorf("s.Mean(): 5151.807692307692 != %v\n", mean)
	}
	if stdDev := s.StdDev(); 3066.508877174006 != stdDev {
		t.Errorf("s.StdDev(): 3066.508877174006 != %v\n", stdDev)
	}
	ps := s.Percentiles([]float64{0.5, 0.75, 0.99})
	if 5237.5 != ps[0] {
		t.Errorf("median: 5237.5 != %v\n", ps[0])
	}
	if 7605.5 != ps[1] {
		t.Errorf("75th percentile: 7605.5 != %v\n", ps[1])
	}
	if 9999.95 != ps[2] {
		t.Errorf("99th percentile: 9999.95 != %v\n", ps[2])
	}
}

var _PRIMES = [...]int64{3, 5, 7, 11, 13, 17, 19, 23, 29, 31, 37, 41, 43, 47, 53, 59, 61, 67}

func setSeed(sample Sample) {
	s := sample.(*ExpDecaySample)
	for r, _ := range s.reservoirs {
		s.reservoirs[r].random = rand.New(rand.NewSource(_PRIMES[r]))
	}
}
