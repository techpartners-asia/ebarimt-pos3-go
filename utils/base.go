package utils

import (
	"fmt"
	"strconv"
)

// int64Pointer Get int64 pointer
func int64Pointer(i int64) *int64 {
	return &i
}

// StrToUint String to Uint parser
func StrToUint(value string) (uint, error) {
	u64, err := strconv.ParseUint(value, 10, 32)
	if err != nil {
		return 0, err
	}
	result := uint(u64)
	return result, nil
}

func AppendAsString(args ...interface{}) string {
	appendedStr := ""
	for _, arg := range args {
		appendedStr = appendedStr + fmt.Sprintf("%v", arg)
	}

	return appendedStr
}

func GetValidString(source interface{}) string {
	if source == nil {
		return ""
	} else {
		return source.(string)
	}
}

func GetValidFloat(source interface{}) float64 {
	if source == nil {
		return float64(0)
	} else {
		num, _ := strconv.ParseFloat(source.(string), 64)
		return num
	}
}

func Float64ToString(f float64) string {
	return fmt.Sprintf("%.2f", f)
}

func NumberPrecision(value float64) float64 {
	return float64(int(value*100)) / 100
	// return float64(int(value*100)) / 100
}

func GetVat(value float64) float64 {
	return (value / 110) * 100000 / 10000
}

func GetVatWithCityTax(value float64) float64 {
	step1 := (value / 112) * 100000 / 10000
	return step1
}

func GetCityTax(value float64) float64 {
	return (value / 112) * 100000 / 50000
}

func GetCityTaxWithoutVat(value float64) float64 {
	return (value / 102) * 100000 / 50000
}
