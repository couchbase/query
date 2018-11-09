package metrics

import "testing"

func testHistogram10000(t *testing.T, h Histogram) {
	if count := h.Count(); 10000 != count {
		t.Errorf("h.Count(): 10000 != %v\n", count)
	}
	if min := h.Min(); 1 != min {
		t.Errorf("h.Min(): 1 != %v\n", min)
	}
	if max := h.Max(); 10000 != max {
		t.Errorf("h.Max(): 10000 != %v\n", max)
	}
	if mean := h.Mean(); 5000.5 != mean {
		t.Errorf("h.Mean(): 5000.5 != %v\n", mean)
	}
	if stdDev := h.StdDev(); 2886.751331514372 != stdDev {
		t.Errorf("h.StdDev(): 2886.751331514372 != %v\n", stdDev)
	}
	ps := h.Percentiles([]float64{0.5, 0.75, 0.99})
	if 5000.5 != ps[0] {
		t.Errorf("median: 5000.5 != %v\n", ps[0])
	}
	if 7500.75 != ps[1] {
		t.Errorf("75th percentile: 7500.75 != %v\n", ps[1])
	}
	if 9900.99 != ps[2] {
		t.Errorf("99th percentile: 9900.99 != %v\n", ps[2])
	}
}
