package finance

import (
	"errors"
	"math/big"
	"time"
)

var ErrInvalidMortgage = errors.New("invalid mortgage parameters")

type MortgageCalculationInput struct {
	MortgageType        string `json:"mortgageType"`
	CommercialPrincipal string `json:"commercialPrincipal"`
	CommercialRate      string `json:"commercialRate"`
	FundPrincipal       string `json:"fundPrincipal"`
	FundRate            string `json:"fundRate"`
	TermMonths          int    `json:"termMonths"`
	RepaymentMethod     string `json:"repaymentMethod"`
	StartDate           string `json:"startDate"`
}

type PaymentItem struct {
	Period             int    `json:"period"`
	PaymentDate        string `json:"paymentDate"`
	Payment            string `json:"payment"`
	Principal          string `json:"principal"`
	Interest           string `json:"interest"`
	RemainingPrincipal string `json:"remainingPrincipal"`
}

type MortgageCalculationResult struct {
	TotalPrincipal string        `json:"totalPrincipal"`
	FirstPayment   string        `json:"firstPayment"`
	LastPayment    string        `json:"lastPayment"`
	TotalInterest  string        `json:"totalInterest"`
	TotalRepayment string        `json:"totalRepayment"`
	TermMonths     int           `json:"termMonths"`
	Schedule       []PaymentItem `json:"schedule"`
}

type centsPayment struct {
	Period, Payment, Principal, Interest, Remaining int64
	Date                                            string
}

func CalculateMortgage(input MortgageCalculationInput) (MortgageCalculationResult, error) {
	if input.TermMonths <= 0 || input.TermMonths > 600 || (input.RepaymentMethod != "annuity" && input.RepaymentMethod != "equal_principal") {
		return MortgageCalculationResult{}, ErrInvalidMortgage
	}
	start, err := time.Parse("2006-01-02", input.StartDate)
	if err != nil {
		return MortgageCalculationResult{}, ErrInvalidMortgage
	}
	commercial, err := ParseCents(defaultZero(input.CommercialPrincipal))
	if err != nil || commercial < 0 {
		return MortgageCalculationResult{}, ErrInvalidMortgage
	}
	fund, err := ParseCents(defaultZero(input.FundPrincipal))
	if err != nil || fund < 0 {
		return MortgageCalculationResult{}, ErrInvalidMortgage
	}
	switch input.MortgageType {
	case "commercial":
		fund = 0
	case "fund":
		commercial = 0
	case "combined":
	default:
		return MortgageCalculationResult{}, ErrInvalidMortgage
	}
	if commercial+fund <= 0 {
		return MortgageCalculationResult{}, ErrInvalidMortgage
	}

	parts := make([][]centsPayment, 0, 2)
	if commercial > 0 {
		rate, parseErr := parseRatePercent(defaultZero(input.CommercialRate))
		if parseErr != nil {
			return MortgageCalculationResult{}, ErrInvalidMortgage
		}
		parts = append(parts, buildSchedule(commercial, rate, input.TermMonths, input.RepaymentMethod, start))
	}
	if fund > 0 {
		rate, parseErr := parseRatePercent(defaultZero(input.FundRate))
		if parseErr != nil {
			return MortgageCalculationResult{}, ErrInvalidMortgage
		}
		parts = append(parts, buildSchedule(fund, rate, input.TermMonths, input.RepaymentMethod, start))
	}

	merged := make([]centsPayment, input.TermMonths)
	for index := range merged {
		merged[index].Period = int64(index + 1)
		merged[index].Date = paymentDate(start, index+1)
		for _, part := range parts {
			merged[index].Payment += part[index].Payment
			merged[index].Principal += part[index].Principal
			merged[index].Interest += part[index].Interest
			merged[index].Remaining += part[index].Remaining
		}
	}
	return paymentResult(commercial+fund, merged), nil
}

func buildSchedule(principal int64, monthlyRate *big.Rat, months int, method string, start time.Time) []centsPayment {
	items := make([]centsPayment, 0, months)
	remaining := principal
	annuityPayment := int64(0)
	if method == "annuity" {
		if monthlyRate.Sign() == 0 {
			annuityPayment = roundRat(new(big.Rat).SetFrac(big.NewInt(principal), big.NewInt(int64(months))))
		} else {
			factor := ratPow(new(big.Rat).Add(big.NewRat(1, 1), monthlyRate), months)
			numerator := new(big.Rat).Mul(new(big.Rat).SetInt64(principal), new(big.Rat).Mul(monthlyRate, factor))
			annuityPayment = roundRat(new(big.Rat).Quo(numerator, new(big.Rat).Sub(factor, big.NewRat(1, 1))))
		}
	}
	basePrincipal := principal / int64(months)
	remainderCents := principal % int64(months)
	for index := 0; index < months; index++ {
		interest := roundRat(new(big.Rat).Mul(new(big.Rat).SetInt64(remaining), monthlyRate))
		principalPaid := int64(0)
		if method == "annuity" {
			principalPaid = annuityPayment - interest
			if monthlyRate.Sign() == 0 {
				principalPaid = principal / int64(months)
				if int64(index) < principal%int64(months) {
					principalPaid++
				}
			}
		} else {
			principalPaid = basePrincipal
			if int64(index) < remainderCents {
				principalPaid++
			}
		}
		if index == months-1 || principalPaid > remaining {
			principalPaid = remaining
		}
		payment := principalPaid + interest
		remaining -= principalPaid
		items = append(items, centsPayment{Period: int64(index + 1), Date: paymentDate(start, index+1), Payment: payment, Principal: principalPaid, Interest: interest, Remaining: remaining})
	}
	return items
}

func ratPow(value *big.Rat, exponent int) *big.Rat {
	result := big.NewRat(1, 1)
	base := new(big.Rat).Set(value)
	for exponent > 0 {
		if exponent%2 == 1 {
			result.Mul(result, base)
		}
		base.Mul(base, base)
		exponent /= 2
	}
	return result
}

func paymentDate(start time.Time, months int) string {
	year, month, day := start.Date()
	monthIndex := int(month) - 1 + months
	targetYear := year + monthIndex/12
	targetMonth := time.Month(monthIndex%12 + 1)
	lastDay := time.Date(targetYear, targetMonth+1, 0, 0, 0, 0, 0, start.Location()).Day()
	if day > lastDay {
		day = lastDay
	}
	return time.Date(targetYear, targetMonth, day, 0, 0, 0, 0, start.Location()).Format("2006-01-02")
}

func paymentResult(principal int64, schedule []centsPayment) MortgageCalculationResult {
	result := MortgageCalculationResult{TotalPrincipal: FormatCents(principal), TermMonths: len(schedule), Schedule: make([]PaymentItem, 0, len(schedule))}
	totalInterest := int64(0)
	for _, item := range schedule {
		totalInterest += item.Interest
		result.Schedule = append(result.Schedule, PaymentItem{Period: int(item.Period), PaymentDate: item.Date, Payment: FormatCents(item.Payment), Principal: FormatCents(item.Principal), Interest: FormatCents(item.Interest), RemainingPrincipal: FormatCents(item.Remaining)})
	}
	if len(schedule) > 0 {
		result.FirstPayment = FormatCents(schedule[0].Payment)
		result.LastPayment = FormatCents(schedule[len(schedule)-1].Payment)
	}
	result.TotalInterest = FormatCents(totalInterest)
	result.TotalRepayment = FormatCents(principal + totalInterest)
	return result
}

func defaultZero(value string) string {
	if value == "" {
		return "0"
	}
	return value
}

type PrepaymentInput struct {
	MortgageCalculationInput
	AfterPeriod      int    `json:"afterPeriod"`
	PrepaymentAmount string `json:"prepaymentAmount"`
	Type             string `json:"type"`
	Strategy         string `json:"strategy"`
}

type PrepaymentResult struct {
	OriginalRemainingInterest string `json:"originalRemainingInterest"`
	NewRemainingInterest      string `json:"newRemainingInterest"`
	InterestSaved             string `json:"interestSaved"`
	NewMonthlyPayment         string `json:"newMonthlyPayment"`
	MonthsSaved               int    `json:"monthsSaved"`
}

// SimulatePrepayment 是测算工具，不包含银行违约金和按日计息规则。
func SimulatePrepayment(input PrepaymentInput) (PrepaymentResult, error) {
	if input.MortgageType == "combined" {
		return PrepaymentResult{}, ErrInvalidMortgage
	}
	original, err := CalculateMortgage(input.MortgageCalculationInput)
	if err != nil || input.AfterPeriod < 0 || input.AfterPeriod >= len(original.Schedule) {
		return PrepaymentResult{}, ErrInvalidMortgage
	}
	amount, err := ParseCents(input.PrepaymentAmount)
	if err != nil || amount <= 0 {
		return PrepaymentResult{}, ErrInvalidMortgage
	}
	remaining, _ := ParseCents(original.TotalPrincipal)
	if input.AfterPeriod > 0 {
		remaining, _ = ParseCents(original.Schedule[input.AfterPeriod-1].RemainingPrincipal)
	}
	if amount > remaining {
		return PrepaymentResult{}, ErrInvalidMortgage
	}
	originalInterest := int64(0)
	for _, item := range original.Schedule[input.AfterPeriod:] {
		value, _ := ParseCents(item.Interest)
		originalInterest += value
	}
	if input.Type == "settle" || amount == remaining {
		return PrepaymentResult{OriginalRemainingInterest: FormatCents(originalInterest), NewRemainingInterest: "0.00", InterestSaved: FormatCents(originalInterest), NewMonthlyPayment: "0.00", MonthsSaved: len(original.Schedule) - input.AfterPeriod}, nil
	}
	remaining -= amount
	rateValue := input.CommercialRate
	if input.MortgageType == "fund" {
		rateValue = input.FundRate
	}
	monthlyRate, err := parseRatePercent(defaultZero(rateValue))
	if err != nil {
		return PrepaymentResult{}, ErrInvalidMortgage
	}
	remainingMonths := len(original.Schedule) - input.AfterPeriod
	startDate := input.StartDate
	if input.AfterPeriod > 0 {
		startDate = original.Schedule[input.AfterPeriod-1].PaymentDate
	}
	start, _ := time.Parse("2006-01-02", startDate)
	var schedule []centsPayment
	if input.Strategy == "shorten_term" {
		paymentIndex := input.AfterPeriod
		if paymentIndex >= len(original.Schedule) {
			paymentIndex = len(original.Schedule) - 1
		}
		payment, _ := ParseCents(original.Schedule[paymentIndex].Payment)
		schedule = buildFixedPaymentSchedule(remaining, monthlyRate, payment, remainingMonths, start)
	} else if input.Strategy == "lower_payment" {
		schedule = buildSchedule(remaining, monthlyRate, remainingMonths, input.RepaymentMethod, start)
	} else {
		return PrepaymentResult{}, ErrInvalidMortgage
	}
	newInterest := int64(0)
	for _, item := range schedule {
		newInterest += item.Interest
	}
	newPayment := int64(0)
	if len(schedule) > 0 {
		newPayment = schedule[0].Payment
	}
	return PrepaymentResult{OriginalRemainingInterest: FormatCents(originalInterest), NewRemainingInterest: FormatCents(newInterest), InterestSaved: FormatCents(originalInterest - newInterest), NewMonthlyPayment: FormatCents(newPayment), MonthsSaved: remainingMonths - len(schedule)}, nil
}

func buildFixedPaymentSchedule(principal int64, rate *big.Rat, payment int64, maxMonths int, start time.Time) []centsPayment {
	items := make([]centsPayment, 0, maxMonths)
	remaining := principal
	for index := 0; remaining > 0 && index < maxMonths; index++ {
		interest := roundRat(new(big.Rat).Mul(new(big.Rat).SetInt64(remaining), rate))
		principalPaid := payment - interest
		if principalPaid <= 0 {
			break
		}
		if principalPaid > remaining {
			principalPaid = remaining
		}
		remaining -= principalPaid
		items = append(items, centsPayment{Period: int64(index + 1), Date: paymentDate(start, index+1), Payment: principalPaid + interest, Principal: principalPaid, Interest: interest, Remaining: remaining})
	}
	return items
}
