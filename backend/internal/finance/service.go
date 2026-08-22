package finance

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"
)

var (
	ErrInvalidInput = errors.New("invalid finance input")
	ErrNotFound     = errors.New("finance record not found")
)

type Service struct{ db *gorm.DB }

func NewService(db *gorm.DB) *Service { return &Service{db: db} }

type Overview struct {
	TotalAssets       string            `json:"totalAssets"`
	TotalLiabilities  string            `json:"totalLiabilities"`
	NetWorth          string            `json:"netWorth"`
	MonthIncome       string            `json:"monthIncome"`
	MonthExpense      string            `json:"monthExpense"`
	MonthBalance      string            `json:"monthBalance"`
	SavingsRate       *string           `json:"savingsRate"`
	DebtRatio         *string           `json:"debtRatio"`
	AccountCount      int               `json:"accountCount"`
	AssetStructure    []AssetSlice      `json:"assetStructure"`
	NetWorthTrend     []NetWorthPoint   `json:"netWorthTrend"`
	CashFlow          []MonthlyCashFlow `json:"cashFlow"`
	ExpenseCategories []CategoryMetric  `json:"expenseCategories"`
	Upcoming          []UpcomingPayment `json:"upcoming"`
}

type AssetSlice struct {
	Name   string `json:"name"`
	Amount string `json:"amount"`
}
type MonthlyCashFlow struct {
	Month   string `json:"month"`
	Income  string `json:"income"`
	Expense string `json:"expense"`
	Net     string `json:"net"`
}
type NetWorthPoint struct {
	Date        string `json:"date"`
	Assets      string `json:"assets"`
	Liabilities string `json:"liabilities"`
	NetWorth    string `json:"netWorth"`
}
type CategoryMetric struct {
	Name   string `json:"name"`
	Amount string `json:"amount"`
	Color  string `json:"color"`
}
type UpcomingPayment struct {
	ID     uint   `json:"id"`
	Name   string `json:"name"`
	Amount string `json:"amount"`
	Date   string `json:"date"`
	Kind   string `json:"kind"`
	Status string `json:"status"`
}

func (s *Service) Overview(userID uint, now time.Time) (Overview, error) {
	var accounts []FinancialAccount
	var cards []CreditCard
	var loans []Loan
	var mortgages []Mortgage
	if err := s.db.Where("user_id = ? AND archived = ?", userID, false).Order("sort_order, id").Find(&accounts).Error; err != nil {
		return Overview{}, err
	}
	if err := s.db.Where("user_id = ? AND archived = ?", userID, false).Find(&cards).Error; err != nil {
		return Overview{}, err
	}
	if err := s.db.Where("user_id = ? AND archived = ?", userID, false).Find(&loans).Error; err != nil {
		return Overview{}, err
	}
	if err := s.db.Where("user_id = ? AND archived = ?", userID, false).Find(&mortgages).Error; err != nil {
		return Overview{}, err
	}

	assetGroups := map[string]int64{}
	for _, account := range accounts {
		if !account.IncludeInNetWorth {
			continue
		}
		value, err := ParseCents(account.Balance)
		if err != nil {
			return Overview{}, err
		}
		assetGroups[account.AccountType] += value
	}

	monthStart := now.Format("2006-01") + "-01"
	monthEnd := paymentDate(time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()), 1)
	monthEnd = monthEnd[:7] + "-01"
	var monthTransactions []Transaction
	if err := s.db.Where("user_id = ? AND transaction_date >= ? AND transaction_date < ?", userID, monthStart, monthEnd).Find(&monthTransactions).Error; err != nil {
		return Overview{}, err
	}
	metrics, err := CalculateMetrics(accounts, cards, loans, mortgages, monthTransactions)
	if err != nil {
		return Overview{}, err
	}
	result := Overview{TotalAssets: FormatCents(metrics.Assets), TotalLiabilities: FormatCents(metrics.Liabilities), NetWorth: FormatCents(metrics.NetWorth), MonthIncome: FormatCents(metrics.Income), MonthExpense: FormatCents(metrics.Expense), MonthBalance: FormatCents(metrics.Balance), SavingsRate: metrics.SavingsRate, DebtRatio: metrics.DebtRatio, AccountCount: len(accounts), AssetStructure: []AssetSlice{}}
	for _, kind := range []string{"bank", "alipay", "wechat", "cash", "savings", "investment", "other"} {
		if value := assetGroups[kind]; value != 0 {
			result.AssetStructure = append(result.AssetStructure, AssetSlice{Name: accountTypeName(kind), Amount: FormatCents(value)})
		}
	}
	result.CashFlow, err = s.monthlyCashFlow(userID, now, 6)
	if err != nil {
		return Overview{}, err
	}
	result.NetWorthTrend, err = s.netWorthTrend(userID, accounts, metrics.Liabilities)
	if err != nil {
		return Overview{}, err
	}
	result.ExpenseCategories, err = s.expenseCategoryMetrics(userID, monthStart, monthEnd)
	if err != nil {
		return Overview{}, err
	}
	result.Upcoming = upcomingPayments(cards, loans, mortgages, userID, s.db, now)
	return result, nil
}

func (s *Service) netWorthTrend(userID uint, accounts []FinancialAccount, liabilities int64) ([]NetWorthPoint, error) {
	included := map[uint]bool{}
	ids := make([]uint, 0, len(accounts))
	for _, account := range accounts {
		if account.IncludeInNetWorth {
			included[account.ID] = true
			ids = append(ids, account.ID)
		}
	}
	if len(ids) == 0 {
		return []NetWorthPoint{}, nil
	}
	var snapshots []BalanceSnapshot
	if err := s.db.Where("user_id = ? AND account_id IN ?", userID, ids).Order("recorded_at, id").Find(&snapshots).Error; err != nil {
		return nil, err
	}
	balances := map[uint]int64{}
	result := make([]NetWorthPoint, 0)
	currentDate := ""
	appendPoint := func(date string) {
		assets := int64(0)
		for id, value := range balances {
			if included[id] {
				assets += value
			}
		}
		point := NetWorthPoint{Date: date, Assets: FormatCents(assets), Liabilities: FormatCents(liabilities), NetWorth: FormatCents(assets - liabilities)}
		if len(result) > 0 && result[len(result)-1].Date == date {
			result[len(result)-1] = point
		} else {
			result = append(result, point)
		}
	}
	for _, snapshot := range snapshots {
		date := snapshot.RecordedAt.Format("2006-01-02")
		if currentDate != "" && date != currentDate {
			appendPoint(currentDate)
		}
		value, err := ParseCents(snapshot.Balance)
		if err != nil {
			return nil, err
		}
		balances[snapshot.AccountID] = value
		currentDate = date
	}
	if currentDate != "" {
		appendPoint(currentDate)
	}
	if len(result) > 366 {
		result = result[len(result)-366:]
	}
	return result, nil
}

func transactionTotals(items []Transaction) (income, expense int64) {
	for _, item := range items {
		amount, err := ParseCents(item.Amount)
		if err != nil {
			continue
		}
		switch item.Type {
		case "income":
			income += amount
		case "expense":
			expense += amount
		}
	}
	return
}

func (s *Service) monthlyCashFlow(userID uint, now time.Time, months int) ([]MonthlyCashFlow, error) {
	start := time.Date(now.Year(), now.Month()-time.Month(months-1), 1, 0, 0, 0, 0, now.Location())
	var items []Transaction
	if err := s.db.Where("user_id = ? AND transaction_date >= ?", userID, start.Format("2006-01-02")).Find(&items).Error; err != nil {
		return nil, err
	}
	byMonth := map[string][]Transaction{}
	for _, item := range items {
		if len(item.TransactionDate) >= 7 {
			byMonth[item.TransactionDate[:7]] = append(byMonth[item.TransactionDate[:7]], item)
		}
	}
	result := make([]MonthlyCashFlow, 0, months)
	for index := 0; index < months; index++ {
		month := time.Date(start.Year(), start.Month()+time.Month(index), 1, 0, 0, 0, 0, now.Location()).Format("2006-01")
		income, expense := transactionTotals(byMonth[month])
		result = append(result, MonthlyCashFlow{Month: month, Income: FormatCents(income), Expense: FormatCents(expense), Net: FormatCents(income - expense)})
	}
	return result, nil
}

func (s *Service) expenseCategoryMetrics(userID uint, start, end string) ([]CategoryMetric, error) {
	type row struct{ Name, Color, Amount string }
	var rows []row
	err := s.db.Table("transactions").Select("COALESCE(transaction_categories.name, '其他') AS name, COALESCE(transaction_categories.color, '#9299A9') AS color, CAST(SUM(transactions.amount) AS CHAR) AS amount").
		Joins("LEFT JOIN transaction_categories ON transaction_categories.id = transactions.category_id").
		Where("transactions.user_id = ? AND transactions.type = ? AND transactions.transaction_date >= ? AND transactions.transaction_date < ?", userID, "expense", start, end).
		Group("transaction_categories.id, transaction_categories.name, transaction_categories.color").Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	result := make([]CategoryMetric, 0, len(rows))
	for _, row := range rows {
		cents, _ := ParseCents(row.Amount)
		result = append(result, CategoryMetric{Name: row.Name, Color: row.Color, Amount: FormatCents(cents)})
	}
	sort.Slice(result, func(i, j int) bool {
		left, _ := ParseCents(result[i].Amount)
		right, _ := ParseCents(result[j].Amount)
		return left > right
	})
	return result, nil
}

func upcomingPayments(cards []CreditCard, loans []Loan, mortgages []Mortgage, userID uint, db *gorm.DB, now time.Time) []UpcomingPayment {
	items := make([]UpcomingPayment, 0)
	appendItem := func(id uint, name, amount, date, kind string) {
		if date == "" {
			return
		}
		due, err := time.ParseInLocation("2006-01-02", date, now.Location())
		if err != nil {
			return
		}
		days := int(due.Sub(time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())).Hours() / 24)
		if days > 30 {
			return
		}
		status := "normal"
		if days < 0 {
			status = "overdue"
		} else if days == 0 {
			status = "today"
		} else if days <= 7 {
			status = "soon"
		}
		items = append(items, UpcomingPayment{ID: id, Name: name, Amount: amount, Date: date, Kind: kind, Status: status})
	}
	for _, item := range cards {
		appendItem(item.ID, item.Name, item.StatementAmount, item.NextPaymentDate, "credit_card")
	}
	for _, item := range loans {
		appendItem(item.ID, item.Name, item.MonthlyPayment, item.NextPaymentDate, "loan")
	}
	for _, item := range mortgages {
		appendItem(item.ID, item.Name, item.MonthlyPayment, item.NextPaymentDate, "mortgage")
	}
	var recurring []RecurringTransaction
	_ = db.Where("user_id = ? AND active = ?", userID, true).Find(&recurring).Error
	for _, item := range recurring {
		appendItem(item.ID, item.Name, item.Amount, item.NextDate, "recurring")
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Date < items[j].Date })
	return items
}

func (s *Service) ListAccounts(userID uint) ([]FinancialAccount, error) {
	var items []FinancialAccount
	err := s.db.Where("user_id = ? AND archived = ?", userID, false).Order("sort_order, id").Find(&items).Error
	return items, err
}
func (s *Service) CreateAccount(userID uint, item *FinancialAccount) error {
	if strings.TrimSpace(item.Name) == "" || !validAccountType(item.AccountType) {
		return ErrInvalidInput
	}
	if _, err := ParseCents(item.Balance); err != nil {
		return ErrInvalidInput
	}
	if item.AvailableBalance == "" {
		item.AvailableBalance = item.Balance
	}
	if _, err := ParseCents(item.AvailableBalance); err != nil {
		return ErrInvalidInput
	}
	item.UserID = userID
	item.MaskedAccountNumber = maskNumber(item.MaskedAccountNumber)
	if item.Currency == "" {
		item.Currency = "CNY"
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(item).Error; err != nil {
			return err
		}
		return tx.Create(&BalanceSnapshot{UserID: userID, AccountID: item.ID, Balance: item.Balance, RecordedAt: time.Now()}).Error
	})
}
func (s *Service) UpdateAccount(userID, id uint, updates map[string]any) error {
	var item FinancialAccount
	if err := s.db.Where("id=? AND user_id=? AND archived=?", id, userID, false).First(&item).Error; err != nil {
		return ErrNotFound
	}
	allowed := map[string]bool{"name": true, "account_type": true, "institution": true, "masked_account_number": true, "balance": true, "available_balance": true, "include_in_net_worth": true, "sort_order": true, "notes": true}
	clean := map[string]any{}
	balanceChanged := false
	for key, value := range updates {
		snake := camelToSnake(key)
		if !allowed[snake] {
			continue
		}
		if snake == "masked_account_number" {
			value = maskNumber(fmt.Sprint(value))
		}
		if snake == "account_type" && !validAccountType(fmt.Sprint(value)) {
			return ErrInvalidInput
		}
		if snake == "name" && strings.TrimSpace(fmt.Sprint(value)) == "" {
			return ErrInvalidInput
		}
		if snake == "balance" || snake == "available_balance" {
			if _, err := ParseCents(fmt.Sprint(value)); err != nil {
				return ErrInvalidInput
			}
		}
		clean[snake] = value
		balanceChanged = balanceChanged || snake == "balance"
	}
	if len(clean) == 0 {
		return ErrInvalidInput
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&item).Updates(clean).Error; err != nil {
			return err
		}
		if !balanceChanged {
			return nil
		}
		if err := tx.First(&item, id).Error; err != nil {
			return err
		}
		return tx.Create(&BalanceSnapshot{UserID: userID, AccountID: id, Balance: item.Balance, RecordedAt: time.Now()}).Error
	})
}

type ArchiveImpact struct {
	Transactions int64 `json:"transactions"`
	Snapshots    int64 `json:"snapshots"`
}

func (s *Service) AccountImpact(userID, id uint) (ArchiveImpact, error) {
	var item FinancialAccount
	if err := s.db.Where("id=? AND user_id=? AND archived=?", id, userID, false).First(&item).Error; err != nil {
		return ArchiveImpact{}, ErrNotFound
	}
	var impact ArchiveImpact
	s.db.Model(&Transaction{}).Where("user_id=? AND (account_id=? OR target_account_id=?)", userID, id, id).Count(&impact.Transactions)
	s.db.Model(&BalanceSnapshot{}).Where("user_id=? AND account_id=?", userID, id).Count(&impact.Snapshots)
	return impact, nil
}
func (s *Service) ArchiveAccount(userID, id uint) (ArchiveImpact, error) {
	var item FinancialAccount
	if err := s.db.Where("id=? AND user_id=? AND archived=?", id, userID, false).First(&item).Error; err != nil {
		return ArchiveImpact{}, ErrNotFound
	}
	var impact ArchiveImpact
	s.db.Model(&Transaction{}).Where("user_id=? AND (account_id=? OR target_account_id=?)", userID, id, id).Count(&impact.Transactions)
	s.db.Model(&BalanceSnapshot{}).Where("user_id=? AND account_id=?", userID, id).Count(&impact.Snapshots)
	return impact, s.db.Model(&item).Update("archived", true).Error
}

type TransactionFilter struct {
	StartDate, EndDate, Type, Keyword string
	MinAmount, MaxAmount              string
	AccountID, CategoryID             uint
}

func (s *Service) ListTransactions(userID uint, filter TransactionFilter) ([]Transaction, error) {
	query := s.db.Where("user_id=?", userID)
	if filter.StartDate != "" {
		query = query.Where("transaction_date>=?", filter.StartDate)
	}
	if filter.EndDate != "" {
		query = query.Where("transaction_date<=?", filter.EndDate)
	}
	if filter.Type != "" {
		query = query.Where("type=?", filter.Type)
	}
	if filter.AccountID > 0 {
		query = query.Where("account_id=? OR target_account_id=?", filter.AccountID, filter.AccountID)
	}
	if filter.CategoryID > 0 {
		query = query.Where("category_id=?", filter.CategoryID)
	}
	if filter.MinAmount != "" {
		if _, err := ParseCents(filter.MinAmount); err != nil {
			return nil, ErrInvalidInput
		}
		query = query.Where("amount >= ?", filter.MinAmount)
	}
	if filter.MaxAmount != "" {
		if _, err := ParseCents(filter.MaxAmount); err != nil {
			return nil, ErrInvalidInput
		}
		query = query.Where("amount <= ?", filter.MaxAmount)
	}
	if filter.Keyword != "" {
		like := "%" + filter.Keyword + "%"
		query = query.Where("counterparty LIKE ? OR description LIKE ? OR tags LIKE ?", like, like, like)
	}
	var items []Transaction
	err := query.Order("transaction_date DESC, id DESC").Limit(500).Find(&items).Error
	return items, err
}
func (s *Service) CreateTransaction(userID uint, item *Transaction) error {
	amount, err := ParseCents(item.Amount)
	if err != nil || amount <= 0 || !validTransactionType(item.Type) {
		return ErrInvalidInput
	}
	if _, err := time.Parse("2006-01-02", item.TransactionDate); err != nil {
		return ErrInvalidInput
	}
	item.UserID = userID
	return s.db.Transaction(func(tx *gorm.DB) error {
		var source FinancialAccount
		if err := tx.Where("id=? AND user_id=? AND archived=?", item.AccountID, userID, false).First(&source).Error; err != nil {
			return ErrInvalidInput
		}
		if item.Type == "transfer" {
			if (item.TargetAccountID == nil) == (item.TargetCreditCardID == nil) {
				return ErrInvalidInput
			}
			item.CategoryID = nil
		} else {
			item.TargetAccountID = nil
			item.TargetCreditCardID = nil
			if item.CategoryID != nil {
				var category TransactionCategory
				if err := tx.Where("id=? AND user_id=? AND type=?", *item.CategoryID, userID, item.Type).First(&category).Error; err != nil {
					return ErrInvalidInput
				}
			}
		}
		delta := amount
		if item.Type == "expense" || item.Type == "transfer" {
			delta = -amount
		}
		if err := adjustAccount(tx, &source, delta, userID); err != nil {
			return err
		}
		if item.Type == "transfer" {
			if item.TargetAccountID != nil {
				if *item.TargetAccountID == item.AccountID {
					return ErrInvalidInput
				}
				var target FinancialAccount
				if err := tx.Where("id=? AND user_id=? AND archived=?", *item.TargetAccountID, userID, false).First(&target).Error; err != nil {
					return ErrInvalidInput
				}
				if err := adjustAccount(tx, &target, amount, userID); err != nil {
					return err
				}
			} else if item.TargetCreditCardID != nil {
				var card CreditCard
				if err := tx.Where("id=? AND user_id=? AND archived=?", *item.TargetCreditCardID, userID, false).First(&card).Error; err != nil {
					return ErrInvalidInput
				}
				debt, _ := ParseCents(card.CurrentDebt)
				debt -= amount
				if debt < 0 {
					debt = 0
				}
				if err := tx.Model(&card).Update("current_debt", FormatCents(debt)).Error; err != nil {
					return err
				}
			} else {
				return ErrInvalidInput
			}
		}
		return tx.Create(item).Error
	})
}
func adjustAccount(tx *gorm.DB, account *FinancialAccount, delta int64, userID uint) error {
	balance, err := ParseCents(account.Balance)
	if err != nil {
		return err
	}
	balance += delta
	account.Balance = FormatCents(balance)
	if err := tx.Model(account).Updates(map[string]any{"balance": account.Balance, "available_balance": account.Balance}).Error; err != nil {
		return err
	}
	return tx.Create(&BalanceSnapshot{UserID: userID, AccountID: account.ID, Balance: account.Balance, RecordedAt: time.Now()}).Error
}

func (s *Service) ListCategories(userID uint) ([]TransactionCategory, error) {
	if err := s.seedCategories(userID); err != nil {
		return nil, err
	}
	var items []TransactionCategory
	err := s.db.Where("user_id=?", userID).Order("type DESC, id").Find(&items).Error
	return items, err
}
func (s *Service) CreateCategory(userID uint, item *TransactionCategory) error {
	if strings.TrimSpace(item.Name) == "" || (item.Type != "income" && item.Type != "expense") {
		return ErrInvalidInput
	}
	item.UserID = userID
	item.IsDefault = false
	if item.Color == "" {
		item.Color = "#6B7280"
	}
	return s.db.Create(item).Error
}
func (s *Service) seedCategories(userID uint) error {
	defaults := map[string][]string{"income": {"工资", "奖金", "兼职", "投资收益", "退款", "其他收入"}, "expense": {"餐饮", "交通", "购物", "住房", "房贷", "水电燃气", "通讯", "娱乐", "医疗", "教育", "旅行", "保险", "订阅", "数码", "人情", "其他"}}
	colors := []string{"#5658cf", "#d06b3c", "#249c77", "#c45064", "#8b70c8", "#4f8da8"}
	for kind, names := range defaults {
		for index, name := range names {
			item := TransactionCategory{UserID: userID, Name: name, Type: kind, Color: colors[index%len(colors)], IsDefault: true}
			if err := s.db.Where(TransactionCategory{UserID: userID, Name: name, Type: kind}).FirstOrCreate(&item).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Service) ListCards(userID uint) ([]CreditCard, error) {
	var items []CreditCard
	err := s.db.Where("user_id=? AND archived=?", userID, false).Order("id").Find(&items).Error
	return items, err
}
func (s *Service) CreateCard(userID uint, item *CreditCard) error {
	if strings.TrimSpace(item.Name) == "" || !validAmounts(item.CreditLimit, item.CurrentDebt, item.StatementAmount, item.MinimumPayment) {
		return ErrInvalidInput
	}
	item.UserID = userID
	item.MaskedAccountNumber = maskNumber(item.MaskedAccountNumber)
	return s.db.Create(item).Error
}
func (s *Service) ListLoans(userID uint) ([]Loan, error) {
	var items []Loan
	err := s.db.Where("user_id=? AND archived=?", userID, false).Order("id").Find(&items).Error
	return items, err
}
func (s *Service) CreateLoan(userID uint, item *Loan) error {
	if strings.TrimSpace(item.Name) == "" || item.TermMonths <= 0 || !validAmounts(item.Principal, item.RemainingPrincipal, item.MonthlyPayment) {
		return ErrInvalidInput
	}
	if _, err := parseRatePercent(defaultZero(item.AnnualInterestRate)); err != nil {
		return ErrInvalidInput
	}
	item.UserID = userID
	return s.db.Create(item).Error
}
func (s *Service) ListMortgages(userID uint) ([]Mortgage, error) {
	var items []Mortgage
	err := s.db.Where("user_id=? AND archived=?", userID, false).Order("id").Find(&items).Error
	return items, err
}
func (s *Service) CreateMortgage(userID uint, item *Mortgage) error {
	input := MortgageCalculationInput{MortgageType: item.MortgageType, CommercialPrincipal: item.CommercialPrincipal, CommercialRate: item.CommercialRate, FundPrincipal: item.FundPrincipal, FundRate: item.FundRate, TermMonths: item.TermMonths, RepaymentMethod: item.RepaymentMethod, StartDate: item.StartDate}
	result, err := CalculateMortgage(input)
	if err != nil || strings.TrimSpace(item.Name) == "" {
		return ErrInvalidInput
	}
	item.UserID = userID
	if item.RemainingPrincipal == "" {
		item.RemainingPrincipal = result.TotalPrincipal
	}
	item.MonthlyPayment = result.FirstPayment
	if item.NextPaymentDate == "" && len(result.Schedule) > 0 {
		item.NextPaymentDate = result.Schedule[0].PaymentDate
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(item).Error; err != nil {
			return err
		}
		rows := make([]MortgagePaymentSchedule, 0, len(result.Schedule))
		for _, p := range result.Schedule {
			rows = append(rows, MortgagePaymentSchedule{MortgageID: item.ID, Period: p.Period, PaymentDate: p.PaymentDate, Payment: p.Payment, Principal: p.Principal, Interest: p.Interest, RemainingPrincipal: p.RemainingPrincipal})
		}
		return tx.CreateInBatches(rows, 100).Error
	})
}

func validAccountType(value string) bool {
	switch value {
	case "bank", "alipay", "wechat", "cash", "savings", "investment", "other":
		return true
	}
	return false
}
func validTransactionType(value string) bool {
	return value == "income" || value == "expense" || value == "transfer"
}
func validAmounts(values ...string) bool {
	for _, value := range values {
		cents, err := ParseCents(defaultZero(value))
		if err != nil || cents < 0 {
			return false
		}
	}
	return true
}
func maskNumber(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, " ", "")
	if value == "" {
		return ""
	}
	if strings.HasPrefix(value, "****") && len(value) <= 9 {
		return value
	}
	runes := []rune(value)
	if len(runes) > 4 {
		runes = runes[len(runes)-4:]
	}
	return "**** " + string(runes)
}
func camelToSnake(value string) string {
	var result strings.Builder
	for index, char := range value {
		if char >= 'A' && char <= 'Z' {
			if index > 0 {
				result.WriteByte('_')
			}
			result.WriteRune(char + ('a' - 'A'))
		} else {
			result.WriteRune(char)
		}
	}
	return result.String()
}
func accountTypeName(value string) string {
	names := map[string]string{"bank": "银行卡", "alipay": "支付宝", "wechat": "微信", "cash": "现金", "savings": "储蓄", "investment": "投资", "other": "其他资产"}
	if name := names[value]; name != "" {
		return name
	}
	return value
}
