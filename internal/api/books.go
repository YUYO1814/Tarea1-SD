package api

import (
	"database/sql"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type Book struct {
	ID              int64  `json:"id"`
	BookName        string `json:"book_name"`
	BookCategory    string `json:"book_category"`
	TransactionType string `json:"transaction_type"` // Venta | Arriendo
	Price           int64  `json:"price"`
	Status          string `json:"status"` // Disponible | Agotado (calculado)
	PopularityScore int64  `json:"popularity_score"`
	Inventory       struct {
		AvailableQuantity int64 `json:"available_quantity"`
	} `json:"inventory"`
}

func registerBookRoutes(r *gin.Engine, db *sql.DB) {
	// POST /books  (crea libro + inventario)
	r.POST("/books", func(c *gin.Context) {
		var in struct {
			BookName          string `json:"book_name"`
			BookCategory      string `json:"book_category"`
			TransactionType   string `json:"transaction_type"` // Venta | Arriendo
			Price             int64  `json:"price"`
			AvailableQuantity int64  `json:"available_quantity"`
		}
		if err := c.BindJSON(&in); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "json inválido"})
			return
		}
		tx, err := db.Begin()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		res, err := tx.Exec(`INSERT INTO books(book_name,book_category,transaction_type,price) VALUES(?,?,?,?)`,
			in.BookName, in.BookCategory, in.TransactionType, in.Price)
		if err != nil {
			tx.Rollback()
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		id, _ := res.LastInsertId()
		if _, err := tx.Exec(`INSERT INTO inventory(book_id,available_quantity) VALUES(?,?)`, id, in.AvailableQuantity); err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		out := Book{
			ID:              id,
			BookName:        in.BookName,
			BookCategory:    in.BookCategory,
			TransactionType: in.TransactionType,
			Price:           in.Price,
		}
		out.Inventory.AvailableQuantity = in.AvailableQuantity
		if in.AvailableQuantity > 0 {
			out.Status = "Disponible"
		} else {
			out.Status = "Agotado"
		}
		c.JSON(http.StatusCreated, out)
	})

	// GET /books  (solo stock > 0, como pide el enunciado)
	r.GET("/books", func(c *gin.Context) {
		rows, err := db.Query(`
SELECT b.id, b.book_name, b.book_category, b.transaction_type, b.price, b.popularity_score, i.available_quantity
FROM books b
JOIN inventory i ON i.book_id = b.id
WHERE i.available_quantity > 0
ORDER BY b.id`)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()

		var list []Book
		for rows.Next() {
			var b Book
			if err := rows.Scan(&b.ID, &b.BookName, &b.BookCategory, &b.TransactionType, &b.Price, &b.PopularityScore, &b.Inventory.AvailableQuantity); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			if b.Inventory.AvailableQuantity > 0 {
				b.Status = "Disponible"
			} else {
				b.Status = "Agotado"
			}
			list = append(list, b)
		}
		c.JSON(http.StatusOK, gin.H{"books": list})
	})

	// PATCH /books/:id  (actualiza precio o cantidad disponible)
	r.PATCH("/books/:id", func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "id inválido"})
			return
		}

		var in struct {
			Price             *int64 `json:"price"`
			AvailableQuantity *int64 `json:"available_quantity"`
		}
		if err := c.BindJSON(&in); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "json inválido"})
			return
		}

		if in.Price != nil {
			if _, err := db.Exec(`UPDATE books SET price=? WHERE id=?`, *in.Price, id); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		}
		if in.AvailableQuantity != nil {
			if _, err := db.Exec(`UPDATE inventory SET available_quantity=? WHERE book_id=?`, *in.AvailableQuantity, id); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		}
		c.Status(http.StatusNoContent)
	})

	// GET /books/popular?limit=10
	r.GET("/books/popular", func(c *gin.Context) {
		limit := 10
		if s := c.DefaultQuery("limit", "10"); s != "" {
			if n, err := strconv.Atoi(s); err == nil && n > 0 && n <= 100 {
				limit = n
			}
		}

		rows, err := db.Query(`
SELECT b.id, b.book_name, b.book_category, b.transaction_type, b.price, b.popularity_score, i.available_quantity
FROM books b
JOIN inventory i ON i.book_id = b.id
ORDER BY b.popularity_score DESC, b.id ASC
LIMIT ?`, limit)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()

		var list []Book
		for rows.Next() {
			var b Book
			if err := rows.Scan(&b.ID, &b.BookName, &b.BookCategory, &b.TransactionType, &b.Price, &b.PopularityScore, &b.Inventory.AvailableQuantity); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			if b.Inventory.AvailableQuantity > 0 {
				b.Status = "Disponible"
			} else {
				b.Status = "Agotado"
			}
			list = append(list, b)
		}
		c.JSON(http.StatusOK, gin.H{"books": list})
	})

}
