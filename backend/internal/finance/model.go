package finance

import "time"

// 所有金额均使用 MySQL DECIMAL 保存，并通过 JSON 字符串传输，避免二进制浮点误差。
type FinancialAccount struct {
	ID                  uint      `json:"id" gorm:"primaryKey"`
	UserID              uint      `json:"-" gorm:"not null;index"`
	Name                string    `json:"name" gorm:"size:128;not null"`
	AccountType         string    `json:"accountType" gorm:"size:32;not null;index"`
	Institution         string    `json:"institution" gorm:"size:128;not null;default:''"`
	MaskedAccountNumber string    `json:"maskedAccountNumber" gorm:"size:32;not null;default:''"`
	Balance             string    `json:"balance" gorm:"type:decimal(20,2);not null;default:0"`
	AvailableBalance    string    `json:"availableBalance" gorm:"type:decimal(20,2);not null;default:0"`
	Currency            string    `json:"currency" gorm:"size:8;not null;default:CNY"`
	IncludeInNetWorth   bool      `json:"includeInNetWorth" gorm:"not null"`
	SortOrder           int       `json:"sortOrder" gorm:"not null;default:0"`
	Notes               string    `json:"notes" gorm:"size:1000;not null;default:''"`
	Archived            bool      `json:"archived" gorm:"not null;default:false;index"`
	CreatedAt           time.Time `json:"createdAt"`
	UpdatedAt           time.Time `json:"updatedAt"`
}

type BalanceSnapshot struct {
	ID         uint      `json:"id" gorm:"primaryKey"`
	UserID     uint      `json:"-" gorm:"not null;index"`
	AccountID  uint      `json:"accountId" gorm:"not null;index"`
	Balance    string    `json:"balance" gorm:"type:decimal(20,2);not null"`
	RecordedAt time.Time `json:"recordedAt" gorm:"not null;index"`
}

type TransactionCategory struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	UserID    uint      `json:"-" gorm:"not null;uniqueIndex:idx_finance_category"`
	Name      string    `json:"name" gorm:"size:64;not null;uniqueIndex:idx_finance_category"`
	Type      string    `json:"type" gorm:"size:16;not null;uniqueIndex:idx_finance_category"`
	Color     string    `json:"color" gorm:"size:20;not null;default:#6B7280"`
	IsDefault bool      `json:"isDefault" gorm:"not null;default:false"`
	CreatedAt time.Time `json:"createdAt"`
}

type Transaction struct {
	ID                 uint      `json:"id" gorm:"primaryKey"`
	UserID             uint      `json:"-" gorm:"not null;index"`
	AccountID          uint      `json:"accountId" gorm:"not null;index"`
	TargetAccountID    *uint     `json:"targetAccountId" gorm:"index"`
	TargetCreditCardID *uint     `json:"targetCreditCardId" gorm:"index"`
	Type               string    `json:"type" gorm:"size:16;not null;index"`
	Amount             string    `json:"amount" gorm:"type:decimal(20,2);not null"`
	CategoryID         *uint     `json:"categoryId" gorm:"index"`
	Counterparty       string    `json:"counterparty" gorm:"size:128;not null;default:''"`
	TransactionDate    string    `json:"transactionDate" gorm:"size:10;not null;index"`
	Description        string    `json:"description" gorm:"size:1000;not null;default:''"`
	Tags               string    `json:"tags" gorm:"size:500;not null;default:''"`
	CreatedAt          time.Time `json:"createdAt"`
	UpdatedAt          time.Time `json:"updatedAt"`
}

type CreditCard struct {
	ID                  uint      `json:"id" gorm:"primaryKey"`
	UserID              uint      `json:"-" gorm:"not null;index"`
	Name                string    `json:"name" gorm:"size:128;not null"`
	Institution         string    `json:"institution" gorm:"size:128;not null;default:''"`
	MaskedAccountNumber string    `json:"maskedAccountNumber" gorm:"size:32;not null;default:''"`
	CreditLimit         string    `json:"creditLimit" gorm:"type:decimal(20,2);not null;default:0"`
	CurrentDebt         string    `json:"currentDebt" gorm:"type:decimal(20,2);not null;default:0"`
	StatementAmount     string    `json:"statementAmount" gorm:"type:decimal(20,2);not null;default:0"`
	MinimumPayment      string    `json:"minimumPayment" gorm:"type:decimal(20,2);not null;default:0"`
	BillingDay          int       `json:"billingDay" gorm:"not null;default:1"`
	RepaymentDay        int       `json:"repaymentDay" gorm:"not null;default:1"`
	NextPaymentDate     string    `json:"nextPaymentDate" gorm:"size:10;not null;default:'';index"`
	HasInstallment      bool      `json:"hasInstallment" gorm:"not null;default:false"`
	Notes               string    `json:"notes" gorm:"size:1000;not null;default:''"`
	Archived            bool      `json:"archived" gorm:"not null;default:false;index"`
	CreatedAt           time.Time `json:"createdAt"`
	UpdatedAt           time.Time `json:"updatedAt"`
}

type Loan struct {
	ID                 uint      `json:"id" gorm:"primaryKey"`
	UserID             uint      `json:"-" gorm:"not null;index"`
	Name               string    `json:"name" gorm:"size:128;not null"`
	LoanType           string    `json:"loanType" gorm:"size:32;not null"`
	Principal          string    `json:"principal" gorm:"type:decimal(20,2);not null"`
	RemainingPrincipal string    `json:"remainingPrincipal" gorm:"type:decimal(20,2);not null"`
	AnnualInterestRate string    `json:"annualInterestRate" gorm:"type:decimal(9,6);not null;default:0"`
	TermMonths         int       `json:"termMonths" gorm:"not null"`
	RepaymentMethod    string    `json:"repaymentMethod" gorm:"size:32;not null"`
	StartDate          string    `json:"startDate" gorm:"size:10;not null"`
	EndDate            string    `json:"endDate" gorm:"size:10;not null;default:''"`
	MonthlyPayment     string    `json:"monthlyPayment" gorm:"type:decimal(20,2);not null;default:0"`
	NextPaymentDate    string    `json:"nextPaymentDate" gorm:"size:10;not null;default:'';index"`
	Lender             string    `json:"lender" gorm:"size:128;not null;default:''"`
	Notes              string    `json:"notes" gorm:"size:1000;not null;default:''"`
	Archived           bool      `json:"archived" gorm:"not null;default:false;index"`
	CreatedAt          time.Time `json:"createdAt"`
	UpdatedAt          time.Time `json:"updatedAt"`
}

type Mortgage struct {
	ID                  uint      `json:"id" gorm:"primaryKey"`
	UserID              uint      `json:"-" gorm:"not null;index"`
	Name                string    `json:"name" gorm:"size:128;not null"`
	MortgageType        string    `json:"mortgageType" gorm:"size:32;not null"`
	CommercialPrincipal string    `json:"commercialPrincipal" gorm:"type:decimal(20,2);not null;default:0"`
	CommercialRate      string    `json:"commercialRate" gorm:"type:decimal(9,6);not null;default:0"`
	FundPrincipal       string    `json:"fundPrincipal" gorm:"type:decimal(20,2);not null;default:0"`
	FundRate            string    `json:"fundRate" gorm:"type:decimal(9,6);not null;default:0"`
	RemainingPrincipal  string    `json:"remainingPrincipal" gorm:"type:decimal(20,2);not null"`
	TermMonths          int       `json:"termMonths" gorm:"not null"`
	RepaymentMethod     string    `json:"repaymentMethod" gorm:"size:32;not null"`
	StartDate           string    `json:"startDate" gorm:"size:10;not null"`
	NextPaymentDate     string    `json:"nextPaymentDate" gorm:"size:10;not null;default:'';index"`
	MonthlyPayment      string    `json:"monthlyPayment" gorm:"type:decimal(20,2);not null;default:0"`
	Notes               string    `json:"notes" gorm:"size:1000;not null;default:''"`
	Archived            bool      `json:"archived" gorm:"not null;default:false;index"`
	CreatedAt           time.Time `json:"createdAt"`
	UpdatedAt           time.Time `json:"updatedAt"`
}

type MortgagePaymentSchedule struct {
	ID                 uint   `json:"id" gorm:"primaryKey"`
	MortgageID         uint   `json:"mortgageId" gorm:"not null;index"`
	Period             int    `json:"period" gorm:"not null"`
	PaymentDate        string `json:"paymentDate" gorm:"size:10;not null"`
	Payment            string `json:"payment" gorm:"type:decimal(20,2);not null"`
	Principal          string `json:"principal" gorm:"type:decimal(20,2);not null"`
	Interest           string `json:"interest" gorm:"type:decimal(20,2);not null"`
	RemainingPrincipal string `json:"remainingPrincipal" gorm:"type:decimal(20,2);not null"`
}

type RecurringTransaction struct {
	ID         uint      `json:"id" gorm:"primaryKey"`
	UserID     uint      `json:"-" gorm:"not null;index"`
	Name       string    `json:"name" gorm:"size:128;not null"`
	Amount     string    `json:"amount" gorm:"type:decimal(20,2);not null"`
	Frequency  string    `json:"frequency" gorm:"size:20;not null"`
	NextDate   string    `json:"nextDate" gorm:"size:10;not null;index"`
	CategoryID *uint     `json:"categoryId"`
	AccountID  *uint     `json:"accountId"`
	Active     bool      `json:"active" gorm:"not null;default:true"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

type Budget struct {
	ID         uint      `json:"id" gorm:"primaryKey"`
	UserID     uint      `json:"-" gorm:"not null;index"`
	Month      string    `json:"month" gorm:"size:7;not null;index"`
	CategoryID *uint     `json:"categoryId" gorm:"index"`
	Amount     string    `json:"amount" gorm:"type:decimal(20,2);not null"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

type FinancialGoal struct {
	ID            uint      `json:"id" gorm:"primaryKey"`
	UserID        uint      `json:"-" gorm:"not null;index"`
	Name          string    `json:"name" gorm:"size:128;not null"`
	TargetAmount  string    `json:"targetAmount" gorm:"type:decimal(20,2);not null"`
	CurrentAmount string    `json:"currentAmount" gorm:"type:decimal(20,2);not null;default:0"`
	TargetDate    string    `json:"targetDate" gorm:"size:10;not null;default:''"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
}
