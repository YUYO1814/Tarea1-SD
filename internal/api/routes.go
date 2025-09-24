package api

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.Engine, db *sql.DB) {
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	registerAuthRoutes(r, db)
	registerUserRoutes(r, db)
	registerBookRoutes(r, db)
	registerSalesRoutes(r, db)
	registerTransactionRoutes(r, db)
	registerLoanRoutes(r, db)

}
