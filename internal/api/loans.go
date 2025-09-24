package api

import (
	"database/sql"
	"math"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type Loan struct {
	ID         int64  `json:"id"`
	UserID     int64  `json:"user_id"`
	BookID     int64  `json:"book_id"`
	StartDate  string `json:"start_date"`
	ReturnDate string `json:"return_date"`
	Status     string `json:"status"`
	DueDate    string `json:"due_date,omitempty"`
	DaysLeft   int64  `json:"days_left,omitempty"`
	DaysLate   int64  `json:"days_late,omitempty"`
	Penalty    int64  `json:"penalty,omitempty"`
}

const loanFmt = "02/01/2006"

func registerLoanRoutes(r *gin.Engine, db *sql.DB) {
	// POST /loans  -> crea préstamo (solo si el libro está en Arriendo y hay stock)
	r.POST("/loans", func(c *gin.Context) {
		var in struct {
			UserID int64 `json:"user_id"`
			BookID int64 `json:"book_id"`
		}
		if err := c.BindJSON(&in); err != nil || in.UserID == 0 || in.BookID == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "json inválido"})
			return
		}

		var kind string
		var qty int64
		if err := db.QueryRow(`
			SELECT b.transaction_type, i.available_quantity
			FROM books b JOIN inventory i ON i.book_id=b.id
			WHERE b.id=?`, in.BookID).Scan(&kind, &qty); err != nil {
			if err == sql.ErrNoRows {
				c.JSON(http.StatusNotFound, gin.H{"error": "libro no existe"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if kind != "Arriendo" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "el libro no está en modalidad Arriendo"})
			return
		}
		if qty <= 0 {
			c.JSON(http.StatusConflict, gin.H{"error": "sin stock"})
			return
		}

		tx, err := db.Begin()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// 1) bajar stock
		res, err := tx.Exec(`UPDATE inventory SET available_quantity=available_quantity-1 WHERE book_id=? AND available_quantity>0`, in.BookID)
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

		// 2) +1 popularidad
		if _, err := tx.Exec(`UPDATE books SET popularity_score=popularity_score+1 WHERE id=?`, in.BookID); err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// 3) crear loan
		start := time.Now().Format(loanFmt)
		res, err = tx.Exec(`INSERT INTO loans(user_id,book_id,start_date,status) VALUES(?,?,?, 'pendiente')`, in.UserID, in.BookID, start)
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

		due := time.Now().AddDate(0, 1, 0)
		out := Loan{
			ID: id, UserID: in.UserID, BookID: in.BookID,
			StartDate: start, Status: "pendiente",
			DueDate:  due.Format(loanFmt),
			DaysLeft: int64(math.Ceil(due.Sub(time.Now()).Hours() / 24)),
		}
		c.JSON(http.StatusCreated, out)
	})

	// GET /loans  -> lista préstamos
	r.GET("/loans", func(c *gin.Context) {
		rows, err := db.Query(`SELECT id,user_id,book_id,start_date,COALESCE(return_date,''),status FROM loans ORDER BY id`)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()

		now := time.Now()
		var out []Loan
		for rows.Next() {
			var l Loan
			if err := rows.Scan(&l.ID, &l.UserID, &l.BookID, &l.StartDate, &l.ReturnDate, &l.Status); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			start, _ := time.ParseInLocation(loanFmt, l.StartDate, time.Local)
			due := start.AddDate(0, 1, 0)
			l.DueDate = due.Format(loanFmt)
			if l.Status == "pendiente" {
				l.DaysLeft = int64(math.Ceil(due.Sub(now).Hours() / 24))
			}
			out = append(out, l)
		}
		c.JSON(http.StatusOK, gin.H{"loans": out})
	})

	// PATCH /loans/:id/return {return_date:"DD/MM/YYYY"}  -> devuelve y multa 2 * días atraso
	r.PATCH("/loans/:id/return", func(c *gin.Context) {
		loanID := c.Param("id")
		var in struct {
			ReturnDate string `json:"return_date"`
		}
		if err := c.BindJSON(&in); err != nil || in.ReturnDate == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "json inválido"})
			return
		}
		ret, err := time.ParseInLocation(loanFmt, in.ReturnDate, time.Local)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "fecha inválida, use DD/MM/YYYY"})
			return
		}

		var userID, bookID int64
		var startStr, status string
		if err := db.QueryRow(`SELECT user_id,book_id,start_date,status FROM loans WHERE id=?`, loanID).
			Scan(&userID, &bookID, &startStr, &status); err != nil {
			if err == sql.ErrNoRows {
				c.JSON(http.StatusNotFound, gin.H{"error": "préstamo no existe"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if status != "pendiente" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "ya devuelto"})
			return
		}

		start, _ := time.ParseInLocation(loanFmt, startStr, time.Local)
		due := start.AddDate(0, 1, 0)
		daysLate := int64(math.Floor(ret.Sub(due).Hours() / 24))
		if daysLate < 0 {
			daysLate = 0
		}
		penalty := daysLate * 2

		tx, err := db.Begin()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// 1) subir stock
		if _, err := tx.Exec(`UPDATE inventory SET available_quantity=available_quantity+1 WHERE book_id=?`, bookID); err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		// 2) cerrar préstamo
		if _, err := tx.Exec(`UPDATE loans SET return_date=?, status='finalizado' WHERE id=?`, in.ReturnDate, loanID); err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		// 3) descontar multa (puede quedar negativo)
		if penalty > 0 {
			if _, err := tx.Exec(`UPDATE users SET usm_pesos = usm_pesos - ? WHERE id=?`, penalty, userID); err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		}

		if err := tx.Commit(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, Loan{
			ID: loanStrToID(loanID), UserID: userID, BookID: bookID,
			StartDate: startStr, ReturnDate: in.ReturnDate, Status: "finalizado",
			DueDate: due.Format(loanFmt), DaysLate: daysLate, Penalty: penalty,
		})
	})
}

func loanStrToID(s string) int64 {
	var x int64
	for _, r := range s {
		if r >= '0' && r <= '9' {
			x = x*10 + int64(r-'0')
		}
	}
	return x
}
