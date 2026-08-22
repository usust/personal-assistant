package finance

import (
	"errors"
	"fmt"
	"math/big"
	"strings"
)

var ErrInvalidMoney = errors.New("invalid money amount")

// ParseCents 将十进制金额严格转换为分，最多接受两位小数。
func ParseCents(value string) (int64, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, ErrInvalidMoney
	}
	negative := strings.HasPrefix(value, "-")
	if negative || strings.HasPrefix(value, "+") {
		value = value[1:]
	}
	parts := strings.Split(value, ".")
	if len(parts) > 2 || parts[0] == "" {
		return 0, ErrInvalidMoney
	}
	frac := ""
	if len(parts) == 2 {
		frac = parts[1]
	}
	if len(frac) > 2 {
		return 0, ErrInvalidMoney
	}
	for len(frac) < 2 {
		frac += "0"
	}
	whole := new(big.Int)
	if _, ok := whole.SetString(parts[0], 10); !ok {
		return 0, ErrInvalidMoney
	}
	minor := new(big.Int)
	if frac == "" {
		frac = "00"
	}
	if _, ok := minor.SetString(frac, 10); !ok {
		return 0, ErrInvalidMoney
	}
	total := new(big.Int).Add(new(big.Int).Mul(whole, big.NewInt(100)), minor)
	if negative {
		total.Neg(total)
	}
	if !total.IsInt64() {
		return 0, ErrInvalidMoney
	}
	return total.Int64(), nil
}

func FormatCents(cents int64) string {
	negative := cents < 0
	if negative {
		cents = -cents
	}
	result := fmt.Sprintf("%d.%02d", cents/100, cents%100)
	if negative {
		return "-" + result
	}
	return result
}

func roundRat(value *big.Rat) int64 {
	if value == nil {
		return 0
	}
	num := new(big.Int).Set(value.Num())
	den := new(big.Int).Set(value.Denom())
	negative := num.Sign() < 0
	if negative {
		num.Abs(num)
	}
	quotient, remainder := new(big.Int), new(big.Int)
	quotient.QuoRem(num, den, remainder)
	if new(big.Int).Mul(remainder, big.NewInt(2)).Cmp(den) >= 0 {
		quotient.Add(quotient, big.NewInt(1))
	}
	if negative {
		quotient.Neg(quotient)
	}
	return quotient.Int64()
}

func parseRatePercent(value string) (*big.Rat, error) {
	rate, ok := new(big.Rat).SetString(strings.TrimSpace(value))
	if !ok || rate.Sign() < 0 {
		return nil, errors.New("invalid annual interest rate")
	}
	return rate.Quo(rate, big.NewRat(1200, 1)), nil
}

func ratioPercent(numerator, denominator int64) *string {
	if denominator == 0 {
		return nil
	}
	hundredths := roundRat(new(big.Rat).SetFrac(new(big.Int).Mul(big.NewInt(numerator), big.NewInt(10000)), big.NewInt(denominator)))
	value := FormatCents(hundredths)
	return &value
}
