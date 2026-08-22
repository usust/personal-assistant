package finance

type Metrics struct {
	Assets      int64
	Liabilities int64
	NetWorth    int64
	Income      int64
	Expense     int64
	Balance     int64
	SavingsRate *string
	DebtRatio   *string
}

// CalculateMetrics 集中定义财务口径：转账不进入收入或支出，归档和不计入净资产的账户被排除。
func CalculateMetrics(accounts []FinancialAccount, cards []CreditCard, loans []Loan, mortgages []Mortgage, transactions []Transaction) (Metrics, error) {
	result := Metrics{}
	for _, account := range accounts {
		if account.Archived || !account.IncludeInNetWorth {
			continue
		}
		value, err := ParseCents(account.Balance)
		if err != nil {
			return Metrics{}, err
		}
		result.Assets += value
	}
	for _, card := range cards {
		if card.Archived {
			continue
		}
		value, err := ParseCents(card.CurrentDebt)
		if err != nil {
			return Metrics{}, err
		}
		result.Liabilities += value
	}
	for _, loan := range loans {
		if loan.Archived {
			continue
		}
		value, err := ParseCents(loan.RemainingPrincipal)
		if err != nil {
			return Metrics{}, err
		}
		result.Liabilities += value
	}
	for _, mortgage := range mortgages {
		if mortgage.Archived {
			continue
		}
		value, err := ParseCents(mortgage.RemainingPrincipal)
		if err != nil {
			return Metrics{}, err
		}
		result.Liabilities += value
	}
	result.Income, result.Expense = transactionTotals(transactions)
	result.NetWorth = result.Assets - result.Liabilities
	result.Balance = result.Income - result.Expense
	if result.Income != 0 {
		result.SavingsRate = ratioPercent(result.Balance, result.Income)
	}
	if result.Assets != 0 {
		result.DebtRatio = ratioPercent(result.Liabilities, result.Assets)
	}
	return result, nil
}

func CreditUtilization(debt, limit string) (*string, error) {
	debtCents, err := ParseCents(debt)
	if err != nil {
		return nil, err
	}
	limitCents, err := ParseCents(limit)
	if err != nil {
		return nil, err
	}
	if limitCents == 0 {
		return nil, nil
	}
	return ratioPercent(debtCents, limitCents), nil
}
