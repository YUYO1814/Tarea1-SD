package api

import (
	"database/sql"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type User struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
	Password  string `json:"password"`
	USMPesos  int64  `json:"usm_pesos"`
}

func registerUserRoutes(r *gin.Engine, db *sql.DB) {
	r.POST("/users", func(c *gin.Context) {
		var in User
		if err := c.BindJSON(&in); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "json inválido"})
			return
		}
		if in.FirstName == "" || in.LastName == "" || in.Email == "" || in.Password == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "faltan campos"})
			return
		}
		res, err := db.Exec(
			`INSERT INTO users(first_name,last_name,email,password,usm_pesos) VALUES(?,?,?,?,0)`,
			in.FirstName, in.LastName, in.Email, in.Password,
		)
		if err != nil {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		id, _ := res.LastInsertId()
		in.ID = id
		in.USMPesos = 0
		c.JSON(http.StatusCreated, in)
	})

	r.GET("/users", func(c *gin.Context) {
		rows, err := db.Query(`SELECT id,first_name,last_name,email,password,usm_pesos FROM users ORDER BY id`)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()

		out := []User{}
		for rows.Next() {
			var u User
			if err := rows.Scan(&u.ID, &u.FirstName, &u.LastName, &u.Email, &u.Password, &u.USMPesos); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			out = append(out, u)
		}
		c.JSON(http.StatusOK, gin.H{"users": out})
	})

	r.GET("/users/:id", func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "id inválido"})
			return
		}
		var u User
		err = db.QueryRow(`SELECT id,first_name,last_name,email,password,usm_pesos FROM users WHERE id=?`, id).
			Scan(&u.ID, &u.FirstName, &u.LastName, &u.Email, &u.Password, &u.USMPesos)
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "no encontrado"})
			return
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, u)
	})
	r.PATCH("/users/:id", func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "id inválido"})
			return
		}

		// Campos opcionales
		var in struct {
			FirstName *string `json:"first_name"`
			LastName  *string `json:"last_name"`
			Password  *string `json:"password"`
			Abonar    *int64  `json:"abonar"` // suma a usm_pesos
		}
		if err := c.BindJSON(&in); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "json inválido"})
			return
		}

		// Actualizaciones simples
		if in.FirstName != nil {
			if _, err := db.Exec(`UPDATE users SET first_name=? WHERE id=?`, *in.FirstName, id); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		}
		if in.LastName != nil {
			if _, err := db.Exec(`UPDATE users SET last_name=? WHERE id=?`, *in.LastName, id); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		}
		if in.Password != nil {
			if _, err := db.Exec(`UPDATE users SET password=? WHERE id=?`, *in.Password, id); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		}
		if in.Abonar != nil {
			if _, err := db.Exec(`UPDATE users SET usm_pesos = usm_pesos + ? WHERE id=?`, *in.Abonar, id); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
		}

		// Devuelve el usuario actualizado
		var u User
		err = db.QueryRow(`SELECT id,first_name,last_name,email,password,usm_pesos FROM users WHERE id=?`, id).
			Scan(&u.ID, &u.FirstName, &u.LastName, &u.Email, &u.Password, &u.USMPesos)
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "no encontrado"})
			return
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, u)
	})

}
