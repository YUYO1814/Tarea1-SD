package api

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
)

func registerAuthRoutes(r *gin.Engine, db *sql.DB) {
	type loginReq struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	r.POST("/login", func(c *gin.Context) {
		var in loginReq
		if err := c.BindJSON(&in); err != nil || in.Email == "" || in.Password == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "credenciales inválidas"})
			return
		}
		var u User
		err := db.QueryRow(`SELECT id,first_name,last_name,email,password,usm_pesos FROM users WHERE email=? AND password=?`,
			in.Email, in.Password).
			Scan(&u.ID, &u.FirstName, &u.LastName, &u.Email, &u.Password, &u.USMPesos)
		if err == sql.ErrNoRows {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "email o contraseña incorrectos"})
			return
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, u)
	})
}
