package api

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Transaction struct {
	ID     int64  `json:"id"`
	Type   string `json:"type"` // Venta | Arriendo
	UserID int64  `json:"user_id"`
	BookID int64  `json:"book_id"`
	Date   string `json:"date"` // DD/MM/YYYY
}

func registerTransactionRoutes(r *gin.Engine, db *sql.DB) {
	// Todas las transacciones (ventas + préstamos)
	r.GET("/transactions", func(c *gin.Context) {
		rows, err := db.Query(`
SELECT id, type, user_id, book_id, date FROM (
  SELECT id, 'Venta'    AS type, user_id, book_id, sale_date  AS date FROM sales
  UNION ALL
  SELECT id, 'Arriendo' AS type, user_id, book_id, start_date AS date FROM loans
)
ORDER BY
  substr(date, 7, 4) || '-' || substr(date, 4, 2) || '-' || substr(date, 1, 2),
  id
`)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()

		var out []Transaction
		for rows.Next() {
			var t Transaction
			if err := rows.Scan(&t.ID, &t.Type, &t.UserID, &t.BookID, &t.Date); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			out = append(out, t)
		}
		c.JSON(http.StatusOK, gin.H{"transactions": out})
	})

	// Transacciones de un usuario
	r.GET("/users/:id/transactions", func(c *gin.Context) {
		userID := c.Param("id")
		rows, err := db.Query(`
SELECT id, type, user_id, book_id, date FROM (
  SELECT id, 'Venta'    AS type, user_id, book_id, sale_date  AS date FROM sales WHERE user_id = ?
  UNION ALL
  SELECT id, 'Arriendo' AS type, user_id, book_id, start_date AS date FROM loans  WHERE user_id = ?
)
ORDER BY
  substr(date, 7, 4) || '-' || substr(date, 4, 2) || '-' || substr(date, 1, 2),
  id
`, userID, userID)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()

		var out []Transaction
		for rows.Next() {
			var t Transaction
			if err := rows.Scan(&t.ID, &t.Type, &t.UserID, &t.BookID, &t.Date); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			out = append(out, t)
		}
		c.JSON(http.StatusOK, gin.H{"transactions": out})
	})
}
