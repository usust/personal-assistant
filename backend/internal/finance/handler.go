package finance

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"personal_assistant_server/internal/middleware"
	"personal_assistant_server/internal/requestlog"
	"personal_assistant_server/internal/response"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func userID(c *gin.Context) (uint, bool) {
	claims, ok := middleware.CurrentClaims(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "请先登录")
		return 0, false
	}
	return claims.UserID, true
}
func pathID(c *gin.Context) (uint, bool) {
	value, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil || value == 0 {
		response.Error(c, http.StatusBadRequest, "资源编号无效")
		return 0, false
	}
	return uint(value), true
}
func bind(c *gin.Context, value any) bool {
	if err := c.ShouldBindJSON(value); err != nil {
		response.Error(c, http.StatusBadRequest, "请求数据格式错误")
		return false
	}
	return true
}
func serviceError(c *gin.Context, message string, err error) {
	switch {
	case errors.Is(err, ErrInvalidInput), errors.Is(err, ErrInvalidMortgage):
		response.Error(c, http.StatusBadRequest, "财务数据无效，请检查金额、日期与关联账户")
	case errors.Is(err, ErrNotFound):
		response.Error(c, http.StatusNotFound, "记录不存在")
	default:
		requestlog.Error(c, message, err)
		response.Error(c, http.StatusInternalServerError, message)
	}
}
func created(c *gin.Context, value any) {
	c.JSON(http.StatusCreated, response.Body{Code: http.StatusCreated, Message: "success", Data: value})
}

func (h *Handler) Overview(c *gin.Context) {
	id, ok := userID(c)
	if !ok {
		return
	}
	result, err := h.service.Overview(id, time.Now())
	if err != nil {
		serviceError(c, "查询财务总览失败", err)
		return
	}
	response.OK(c, result)
}
func (h *Handler) ListAccounts(c *gin.Context) {
	id, ok := userID(c)
	if !ok {
		return
	}
	items, err := h.service.ListAccounts(id)
	if err != nil {
		serviceError(c, "查询账户失败", err)
		return
	}
	response.OK(c, items)
}
func (h *Handler) CreateAccount(c *gin.Context) {
	id, ok := userID(c)
	if !ok {
		return
	}
	var item FinancialAccount
	if !bind(c, &item) {
		return
	}
	if err := h.service.CreateAccount(id, &item); err != nil {
		serviceError(c, "创建账户失败", err)
		return
	}
	created(c, item)
}
func (h *Handler) UpdateAccount(c *gin.Context) {
	user, ok := userID(c)
	if !ok {
		return
	}
	id, ok := pathID(c)
	if !ok {
		return
	}
	var updates map[string]any
	if !bind(c, &updates) {
		return
	}
	if err := h.service.UpdateAccount(user, id, updates); err != nil {
		serviceError(c, "更新账户失败", err)
		return
	}
	response.OK(c, gin.H{"id": id})
}
func (h *Handler) AccountImpact(c *gin.Context) {
	user, ok := userID(c)
	if !ok {
		return
	}
	id, ok := pathID(c)
	if !ok {
		return
	}
	impact, err := h.service.AccountImpact(user, id)
	if err != nil {
		serviceError(c, "查询账户关联数据失败", err)
		return
	}
	response.OK(c, impact)
}
func (h *Handler) ArchiveAccount(c *gin.Context) {
	user, ok := userID(c)
	if !ok {
		return
	}
	id, ok := pathID(c)
	if !ok {
		return
	}
	impact, err := h.service.ArchiveAccount(user, id)
	if err != nil {
		serviceError(c, "归档账户失败", err)
		return
	}
	response.OK(c, impact)
}

func (h *Handler) ListTransactions(c *gin.Context) {
	id, ok := userID(c)
	if !ok {
		return
	}
	account, _ := strconv.ParseUint(c.Query("accountId"), 10, 32)
	category, _ := strconv.ParseUint(c.Query("categoryId"), 10, 32)
	items, err := h.service.ListTransactions(id, TransactionFilter{StartDate: c.Query("startDate"), EndDate: c.Query("endDate"), Type: c.Query("type"), Keyword: c.Query("keyword"), MinAmount: c.Query("minAmount"), MaxAmount: c.Query("maxAmount"), AccountID: uint(account), CategoryID: uint(category)})
	if err != nil {
		serviceError(c, "查询流水失败", err)
		return
	}
	response.OK(c, items)
}
func (h *Handler) CreateTransaction(c *gin.Context) {
	id, ok := userID(c)
	if !ok {
		return
	}
	var item Transaction
	if !bind(c, &item) {
		return
	}
	if err := h.service.CreateTransaction(id, &item); err != nil {
		serviceError(c, "创建流水失败", err)
		return
	}
	created(c, item)
}
func (h *Handler) ListCategories(c *gin.Context) {
	id, ok := userID(c)
	if !ok {
		return
	}
	items, err := h.service.ListCategories(id)
	if err != nil {
		serviceError(c, "查询分类失败", err)
		return
	}
	response.OK(c, items)
}
func (h *Handler) CreateCategory(c *gin.Context) {
	id, ok := userID(c)
	if !ok {
		return
	}
	var item TransactionCategory
	if !bind(c, &item) {
		return
	}
	if err := h.service.CreateCategory(id, &item); err != nil {
		serviceError(c, "创建分类失败", err)
		return
	}
	created(c, item)
}

func (h *Handler) ListCards(c *gin.Context) {
	id, ok := userID(c)
	if !ok {
		return
	}
	items, err := h.service.ListCards(id)
	if err != nil {
		serviceError(c, "查询信用卡失败", err)
		return
	}
	response.OK(c, items)
}
func (h *Handler) CreateCard(c *gin.Context) {
	id, ok := userID(c)
	if !ok {
		return
	}
	var item CreditCard
	if !bind(c, &item) {
		return
	}
	if err := h.service.CreateCard(id, &item); err != nil {
		serviceError(c, "创建信用卡失败", err)
		return
	}
	created(c, item)
}
func (h *Handler) ListLoans(c *gin.Context) {
	id, ok := userID(c)
	if !ok {
		return
	}
	items, err := h.service.ListLoans(id)
	if err != nil {
		serviceError(c, "查询贷款失败", err)
		return
	}
	response.OK(c, items)
}
func (h *Handler) CreateLoan(c *gin.Context) {
	id, ok := userID(c)
	if !ok {
		return
	}
	var item Loan
	if !bind(c, &item) {
		return
	}
	if err := h.service.CreateLoan(id, &item); err != nil {
		serviceError(c, "创建贷款失败", err)
		return
	}
	created(c, item)
}
func (h *Handler) ListMortgages(c *gin.Context) {
	id, ok := userID(c)
	if !ok {
		return
	}
	items, err := h.service.ListMortgages(id)
	if err != nil {
		serviceError(c, "查询房贷失败", err)
		return
	}
	response.OK(c, items)
}
func (h *Handler) CreateMortgage(c *gin.Context) {
	id, ok := userID(c)
	if !ok {
		return
	}
	var item Mortgage
	if !bind(c, &item) {
		return
	}
	if err := h.service.CreateMortgage(id, &item); err != nil {
		serviceError(c, "创建房贷失败", err)
		return
	}
	created(c, item)
}
func (h *Handler) CalculateMortgage(c *gin.Context) {
	if _, ok := userID(c); !ok {
		return
	}
	var input MortgageCalculationInput
	if !bind(c, &input) {
		return
	}
	result, err := CalculateMortgage(input)
	if err != nil {
		serviceError(c, "房贷计算失败", err)
		return
	}
	response.OK(c, result)
}
func (h *Handler) SimulatePrepayment(c *gin.Context) {
	if _, ok := userID(c); !ok {
		return
	}
	var input PrepaymentInput
	if !bind(c, &input) {
		return
	}
	result, err := SimulatePrepayment(input)
	if err != nil {
		serviceError(c, "提前还款测算失败", err)
		return
	}
	response.OK(c, result)
}
