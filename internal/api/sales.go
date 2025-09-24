package api

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type Sale struct {
	ID       int64  `json:"id"`
	UserID   int64  `json:"user_id"`
	BookID   int64  `json:"book_id"`
	SaleDate string `json:"sale_date"` // DD/MM/YYYY
}

func registerSalesRoutes(r *gin.Engine, db *sql.DB) {
	// POST /sales  -> compra un (1) libro
	r.POST("/sales", func(c *gin.Context) {
		var in struct {
			UserID int64 `json:"user_id"`
			BookID int64 `json:"book_id"`
		}
		if err := c.BindJSON(&in); err != nil || in.UserID == 0 || in.BookID == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "json inválido"})
			return
		}

		// 1) obtener info de libro e inventario
		var txType string
		var price, qty int64
		err := db.QueryRow(`
SELECT b.transaction_type, b.price, i.available_quantity
FROM books b
JOIN inventory i ON i.book_id = b.id
WHERE b.id = ?`, in.BookID).Scan(&txType, &price, &qty)
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "libro no existe"})
			return
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if txType != "Venta" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "el libro no está en modalidad Venta"})
			return
		}
		if qty <= 0 {
			c.JSON(http.StatusConflict, gin.H{"error": "sin stock"})
			return
		}

		// 2) verificar fondos del usuario
		var saldo int64
		if err := db.QueryRow(`SELECT usm_pesos FROM users WHERE id=?`, in.UserID).Scan(&saldo); err != nil {
			if err == sql.ErrNoRows {
				c.JSON(http.StatusNotFound, gin.H{"error": "usuario no existe"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if saldo < price {
			c.JSON(http.StatusBadRequest, gin.H{"error": "fondos insuficientes"})
			return
		}

		// 3) ejecutar transacción: descuenta saldo, stock, aumenta popularidad e inserta venta
		tx, err := db.Begin()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		if _, err := tx.Exec(`UPDATE users SET usm_pesos = usm_pesos - ? WHERE id=?`, price, in.UserID); err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		res, err := tx.Exec(`UPDATE inventory SET available_quantity = available_quantity - 1 WHERE book_id=? AND available_quantity > 0`, in.BookID)
		if err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if n, _ := res.RowsAffected(); n == 0 {
			tx.Rollback()
			c.JSON(http.StatusConflict, gin.H{"error": "sin stock"})
			return
		}
		if _, err := tx.Exec(`UPDATE books SET popularity_score = popularity_score + 1 WHERE id=?`, in.BookID); err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		date := time.Now().Format("02/01/2006")
		res, err = tx.Exec(`INSERT INTO sales(user_id, book_id, sale_date) VALUES(?,?,?)`, in.UserID, in.BookID, date)
		if err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		id, _ := res.LastInsertId()

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, Sale{ID: id, UserID: in.UserID, BookID: in.BookID, SaleDate: date})
	})

	// GET /sales  -> lista ventas
	r.GET("/sales", func(c *gin.Context) {
		rows, err := db.Query(`SELECT id, user_id, book_id, sale_date FROM sales ORDER BY id`)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()
		var out []Sale
		for rows.Next() {
			var s Sale
			if err := rows.Scan(&s.ID, &s.UserID, &s.BookID, &s.SaleDate); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			out = append(out, s)
		}
		c.JSON(http.StatusOK, gin.H{"sales": out})
	})
}
