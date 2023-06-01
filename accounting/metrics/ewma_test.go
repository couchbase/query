package metrics

import (
	"math"
	"testing"
)

func BenchmarkEWMA(b *testing.B) {
	a := NewEWMA1()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		a.Update(1)
		a.Tick()
	}
}

func TestEWMA1(t *testing.T) {
	a := NewEWMA1()
	a.Update(3)
	a.Tick()
	if rate := a.Rate(); 0.6 != rate {
		t.Errorf("initial a.Rate(): 0.6 != %v\n", rate)
	}
	elapseMinute(a)
	if rate := a.Rate(); 0.22072766470286553 != rate {
		t.Errorf("1 minute a.Rate(): 0.22072766470286553 != %v\n", rate)
	}
	elapseMinute(a)
	if rate := a.Rate(); 0.08120116994196772 != rate {
		t.Errorf("2 minute a.Rate(): 0.08120116994196772 != %v\n", rate)
	}
	elapseMinute(a)
	if rate := a.Rate(); 0.0298722410207184 != roundFloat(rate, 16) {
		t.Errorf("3 minute a.Rate(): 0.0298722410207184 != %v\n", roundFloat(rate, 16))
	}
	elapseMinute(a)
	if rate := a.Rate(); 0.0109893833332405 != roundFloat(rate, 16) {
		t.Errorf("4 minute a.Rate(): 0.0109893833332405 != %v\n", roundFloat(rate, 16))
	}
	elapseMinute(a)
	if rate := a.Rate(); 0.00404276819945129 != roundFloat(rate, 17) {
		t.Errorf("5 minute a.Rate(): 0.00404276819945129 != %v\n", roundFloat(rate, 17))
	}
	elapseMinute(a)
	if rate := a.Rate(); 0.00148725130599982 != roundFloat(rate, 17) {
		t.Errorf("6 minute a.Rate(): 0.00148725130599982 != %v\n", roundFloat(rate, 17))
	}
	elapseMinute(a)
	if rate := a.Rate(); 0.000547129179332712 != roundFloat(rate, 18) {
		t.Errorf("7 minute a.Rate(): 0.000547129179332712 != %v\n", roundFloat(rate, 18))
	}
	elapseMinute(a)
	if rate := a.Rate(); 0.000201277576741508 != roundFloat(rate, 18) {
		t.Errorf("8 minute a.Rate(): 0.000201277576741508 != %v\n", roundFloat(rate, 18))
	}
	elapseMinute(a)
	if rate := a.Rate(); 7.40458824520081e-05 != roundFloat(rate, 19) {
		t.Errorf("9 minute a.Rate(): 7.40458824520081e-05 != %v\n", roundFloat(rate, 19))
	}
	elapseMinute(a)
	if rate := a.Rate(); 2.7239957857491083e-05 != rate {
		t.Errorf("10 minute a.Rate(): 2.7239957857491083e-05 != %v\n", rate)
	}
	elapseMinute(a)
	if rate := a.Rate(); 1.00210204741475e-05 != roundFloat(rate, 19) {
		t.Errorf("11 minute a.Rate(): 1.00210204741475e-05 != %v\n", roundFloat(rate, 19))
	}
	elapseMinute(a)
	if rate := a.Rate(); 3.686527411997e-06 != roundFloat(rate, 19) {
		t.Errorf("12 minute a.Rate(): 3.686527411997e-06 != %v\n", roundFloat(rate, 19))
	}
	elapseMinute(a)
	if rate := a.Rate(); 1.35619764418864e-06 != roundFloat(rate, 20) {
		t.Errorf("13 minute a.Rate(): 1.35619764418864e-06 != %v\n", roundFloat(rate, 20))
	}
	elapseMinute(a)
	if rate := a.Rate(); 4.989172314621e-07 != roundFloat(rate, 19) {
		t.Errorf("14 minute a.Rate(): 4.989172314621e-07 != %v\n", roundFloat(rate, 19))
	}
	elapseMinute(a)
	if rate := a.Rate(); 1.8354139230109722e-07 != rate {
		t.Errorf("15 minute a.Rate(): 1.8354139230109722e-07 != %v\n", rate)
	}
}

func TestEWMA5(t *testing.T) {
	a := NewEWMA5()
	a.Update(3)
	a.Tick()
	if rate := a.Rate(); 0.6 != rate {
		t.Errorf("initial a.Rate(): 0.6 != %v\n", rate)
	}
	elapseMinute(a)
	if rate := a.Rate(); 0.49123845184678905 != rate {
		t.Errorf("1 minute a.Rate(): 0.49123845184678905 != %v\n", rate)
	}
	elapseMinute(a)
	if rate := a.Rate(); 0.4021920276213837 != rate {
		t.Errorf("2 minute a.Rate(): 0.4021920276213837 != %v\n", rate)
	}
	elapseMinute(a)
	if rate := a.Rate(); 0.32928698165641596 != rate {
		t.Errorf("3 minute a.Rate(): 0.32928698165641596 != %v\n", rate)
	}
	elapseMinute(a)
	if rate := a.Rate(); 0.269597378470333 != rate {
		t.Errorf("4 minute a.Rate(): 0.269597378470333 != %v\n", rate)
	}
	elapseMinute(a)
	if rate := a.Rate(); 0.2207276647028654 != rate {
		t.Errorf("5 minute a.Rate(): 0.2207276647028654 != %v\n", rate)
	}
	elapseMinute(a)
	if rate := a.Rate(); 0.18071652714732128 != rate {
		t.Errorf("6 minute a.Rate(): 0.18071652714732128 != %v\n", rate)
	}
	elapseMinute(a)
	if rate := a.Rate(); 0.1479581783649639 != roundFloat(rate, 16) {
		t.Errorf("7 minute a.Rate(): 0.1479581783649639 != %v\n", roundFloat(rate, 16))
	}
	elapseMinute(a)
	if rate := a.Rate(); 0.12113791079679326 != rate {
		t.Errorf("8 minute a.Rate(): 0.12113791079679326 != %v\n", rate)
	}
	elapseMinute(a)
	if rate := a.Rate(); 0.09917933293295193 != rate {
		t.Errorf("9 minute a.Rate(): 0.09917933293295193 != %v\n", rate)
	}
	elapseMinute(a)
	if rate := a.Rate(); 0.08120116994196763 != rate {
		t.Errorf("10 minute a.Rate(): 0.08120116994196763 != %v\n", rate)
	}
	elapseMinute(a)
	if rate := a.Rate(); 0.06648189501740036 != rate {
		t.Errorf("11 minute a.Rate(): 0.06648189501740036 != %v\n", rate)
	}
	elapseMinute(a)
	if rate := a.Rate(); 0.0544307719736475 != roundFloat(rate, 16) {
		t.Errorf("12 minute a.Rate(): 0.0544307719736475 != %v\n", roundFloat(rate, 16))
	}
	elapseMinute(a)
	if rate := a.Rate(); 0.0445641469286004 != roundFloat(rate, 16) {
		t.Errorf("13 minute a.Rate(): 0.0445641469286004 != %v\n", roundFloat(rate, 16))
	}
	elapseMinute(a)
	if rate := a.Rate(); 0.0364860375751308 != roundFloat(rate, 16) {
		t.Errorf("14 minute a.Rate(): 0.0364860375751308 != %v\n", roundFloat(rate, 16))
	}
	elapseMinute(a)
	if rate := a.Rate(); 0.0298722410207183831020718428 != rate {
		t.Errorf("15 minute a.Rate(): 0.0298722410207183831020718428 != %v\n", rate)
	}
}

func TestEWMA15(t *testing.T) {
	a := NewEWMA15()
	a.Update(3)
	a.Tick()
	if rate := a.Rate(); 0.6 != rate {
		t.Errorf("initial a.Rate(): 0.6 != %v\n", rate)
	}
	elapseMinute(a)
	if rate := a.Rate(); 0.5613041910189706 != rate {
		t.Errorf("1 minute a.Rate(): 0.5613041910189706 != %v\n", rate)
	}
	elapseMinute(a)
	if rate := a.Rate(); 0.5251039914257684 != rate {
		t.Errorf("2 minute a.Rate(): 0.5251039914257684 != %v\n", rate)
	}
	elapseMinute(a)
	if rate := a.Rate(); 0.4912384518467888184678905 != rate {
		t.Errorf("3 minute a.Rate(): 0.4912384518467888184678905 != %v\n", rate)
	}
	elapseMinute(a)
	if rate := a.Rate(); 0.459557003018789 != rate {
		t.Errorf("4 minute a.Rate(): 0.459557003018789 != %v\n", rate)
	}
	elapseMinute(a)
	if rate := a.Rate(); 0.4299187863442732 != rate {
		t.Errorf("5 minute a.Rate(): 0.4299187863442732 != %v\n", rate)
	}
	elapseMinute(a)
	if rate := a.Rate(); 0.4021920276213831 != rate {
		t.Errorf("6 minute a.Rate(): 0.4021920276213831 != %v\n", rate)
	}
	elapseMinute(a)
	if rate := a.Rate(); 0.37625345116383313 != rate {
		t.Errorf("7 minute a.Rate(): 0.37625345116383313 != %v\n", rate)
	}
	elapseMinute(a)
	if rate := a.Rate(); 0.3519877317060185 != rate {
		t.Errorf("8 minute a.Rate(): 0.3519877317060185 != %v\n", rate)
	}
	elapseMinute(a)
	if rate := a.Rate(); 0.3292869816564153165641596 != rate {
		t.Errorf("9 minute a.Rate(): 0.3292869816564153165641596 != %v\n", rate)
	}
	elapseMinute(a)
	if rate := a.Rate(); 0.3080502714195546 != rate {
		t.Errorf("10 minute a.Rate(): 0.3080502714195546 != %v\n", rate)
	}
	elapseMinute(a)
	if rate := a.Rate(); 0.2881831806538789 != rate {
		t.Errorf("11 minute a.Rate(): 0.2881831806538789 != %v\n", rate)
	}
	elapseMinute(a)
	if rate := a.Rate(); 0.26959737847033216 != rate {
		t.Errorf("12 minute a.Rate(): 0.26959737847033216 != %v\n", rate)
	}
	elapseMinute(a)
	if rate := a.Rate(); 0.2522102307052083 != rate {
		t.Errorf("13 minute a.Rate(): 0.2522102307052083 != %v\n", rate)
	}
	elapseMinute(a)
	if rate := a.Rate(); 0.23594443252115815 != rate {
		t.Errorf("14 minute a.Rate(): 0.23594443252115815 != %v\n", rate)
	}
	elapseMinute(a)
	if rate := a.Rate(); 0.2207276647028646247028654470286553 != rate {
		t.Errorf("15 minute a.Rate(): 0.2207276647028646247028654470286553 != %v\n", rate)
	}
}

func elapseMinute(a EWMA) {
	for i := 0; i < 12; i++ {
		a.Tick()
	}
}

func roundFloat(f float64, precision uint) float64 {
	factor := math.Pow(10, float64(precision))
	return math.Round(f*factor) / factor
}
