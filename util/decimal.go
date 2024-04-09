package util

import (
	"fmt"
	"strconv"
)

// DecimalFromFloat32 Keep 2 bit after decimal point
func DecimalFromFloat32(f float32) float32 {
	res, err := strconv.ParseFloat(fmt.Sprintf("%.2f", f), 32)
	if err != nil {
		return 0
	}
	return float32(res)
}

// DecimailFromFloat64 Keep 2 bit after decimal point
func DecimailFromFloat64(f float64) float64 {
	res, err := strconv.ParseFloat(fmt.Sprintf("%.2f", f), 64)
	if err != nil {
		return 0
	}
	return res
}
