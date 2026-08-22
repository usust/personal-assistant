package finance

import "github.com/gin-gonic/gin"

func RegisterRoutes(router *gin.RouterGroup, handler *Handler) {
	group := router.Group("/finance")
	group.GET("/overview", handler.Overview)
	group.GET("/accounts", handler.ListAccounts)
	group.POST("/accounts", handler.CreateAccount)
	group.PATCH("/accounts/:id", handler.UpdateAccount)
	group.GET("/accounts/:id/impact", handler.AccountImpact)
	group.DELETE("/accounts/:id", handler.ArchiveAccount)
	group.GET("/transactions", handler.ListTransactions)
	group.POST("/transactions", handler.CreateTransaction)
	group.GET("/categories", handler.ListCategories)
	group.POST("/categories", handler.CreateCategory)
	group.GET("/credit-cards", handler.ListCards)
	group.POST("/credit-cards", handler.CreateCard)
	group.GET("/loans", handler.ListLoans)
	group.POST("/loans", handler.CreateLoan)
	group.GET("/mortgages", handler.ListMortgages)
	group.POST("/mortgages", handler.CreateMortgage)
	group.POST("/mortgage/calculate", handler.CalculateMortgage)
	group.POST("/mortgage/prepayment", handler.SimulatePrepayment)
}
