package svdom

import (
	"math"
	"math/big"
)

// Python number semantics for pyjson values: integers (bool counts as 0/1) stay exact
// integers, any float operand gives a float, as `int * float` does in Python.

// bigInt returns an integer value (bool included) as big.Int.
func bigInt(value any) (*big.Int, bool) {
	switch v := value.(type) {
	case bool:
		if v {
			return big.NewInt(1), true
		}
		return big.NewInt(0), true
	case int:
		return big.NewInt(int64(v)), true
	case int64:
		return big.NewInt(v), true
	case *big.Int:
		return new(big.Int).Set(v), true
	}
	return nil, false
}

// toFloat returns a number as float64 (Python float(value)).
func toFloat(value any) (float64, bool) {
	if f, ok := value.(float64); ok {
		return f, true
	}
	if n, ok := bigInt(value); ok {
		f, _ := new(big.Float).SetInt(n).Float64()
		return f, true
	}
	return 0, false
}

// intResult turns an exact integer into the pyjson integer form (int64 or *big.Int).
func intResult(n *big.Int) any {
	if n.IsInt64() {
		return n.Int64()
	}
	return n
}

// mul is a * b; nil when either is not a number (Python would raise or repeat a sequence).
func mul(a, b any) any {
	x, xInt := bigInt(a)
	y, yInt := bigInt(b)
	if xInt && yInt {
		return intResult(new(big.Int).Mul(x, y))
	}
	fa, okA := toFloat(a)
	fb, okB := toFloat(b)
	if !okA || !okB {
		return nil
	}
	return fa * fb
}

// mod is a % b with Python's sign of the divisor; ok is false on a zero divisor or a
// non-number (ZeroDivisionError or TypeError in Python).
func mod(a, b any) (any, bool) {
	x, xInt := bigInt(a)
	y, yInt := bigInt(b)
	if xInt && yInt {
		if y.Sign() == 0 {
			return nil, false
		}
		_, m := floorDivMod(x, y)
		return intResult(m), true
	}
	fa, okA := toFloat(a)
	fb, okB := toFloat(b)
	if !okA || !okB || fb == 0 {
		return nil, false
	}
	_, m := floatDivMod(fa, fb)
	return m, true
}

// floorDiv is a // b; ok as for mod.
func floorDiv(a, b any) (any, bool) {
	x, xInt := bigInt(a)
	y, yInt := bigInt(b)
	if xInt && yInt {
		if y.Sign() == 0 {
			return nil, false
		}
		q, _ := floorDivMod(x, y)
		return intResult(q), true
	}
	fa, okA := toFloat(a)
	fb, okB := toFloat(b)
	if !okA || !okB || fb == 0 {
		return nil, false
	}
	q, _ := floatDivMod(fa, fb)
	return q, true
}

// floorDivMod is integer divmod rounding toward negative infinity.
func floorDivMod(x, y *big.Int) (*big.Int, *big.Int) {
	q, m := new(big.Int).QuoRem(x, y, new(big.Int))
	if m.Sign() != 0 && (m.Sign() < 0) != (y.Sign() < 0) {
		q.Sub(q, big.NewInt(1))
		m.Add(m, y)
	}
	return q, m
}

// floatDivMod is CPython's float_divmod.
func floatDivMod(vx, wx float64) (float64, float64) {
	m := math.Mod(vx, wx)
	div := (vx - m) / wx
	if m != 0 {
		if (wx < 0) != (m < 0) {
			m += wx
			div -= 1.0
		}
	} else {
		m = math.Copysign(0, wx)
	}
	var floorDiv float64
	if div != 0 {
		floorDiv = math.Floor(div)
		if div-floorDiv > 0.5 {
			floorDiv += 1.0
		}
	} else {
		floorDiv = math.Copysign(0, vx/wx)
	}
	return floorDiv, m
}

// compare returns -1, 0 or 1 for two numbers; ok is false when either is not a number
// or is NaN (Python comparisons with NaN are all False).
func compare(a, b any) (int, bool) {
	x, xOK := exact(a)
	y, yOK := exact(b)
	if !xOK || !yOK {
		return 0, false
	}
	return x.Cmp(y), true
}

func exact(value any) (*big.Float, bool) {
	if f, isFloat := value.(float64); isFloat {
		if math.IsNaN(f) {
			return nil, false
		}
		return new(big.Float).SetFloat64(f), true
	}
	if n, ok := bigInt(value); ok {
		return new(big.Float).SetInt(n), true
	}
	return nil, false
}

// greater is Python's a > b for numbers (false for non-numbers).
func greater(a, b any) bool {
	order, ok := compare(a, b)
	return ok && order > 0
}
