package mhandlers

import (
	"github.com/gin-gonic/gin"
	"github.com/pritunl/pritunl-zero/auth"
	"github.com/pritunl/pritunl-zero/cookie"
	"github.com/pritunl/pritunl-zero/database"
	"github.com/pritunl/pritunl-zero/errortypes"
	"github.com/pritunl/pritunl-zero/session"
	"strings"
)

func authStateGet(c *gin.Context) {
	data := auth.GetState()
	c.JSON(200, data)
}

type authData struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func authSessionPost(c *gin.Context) {
	db := c.MustGet("db").(*database.Database)
	data := &authData{}

	err := c.Bind(data)
	if err != nil {
		c.AbortWithError(500, err)
		return
	}

	usr, errData, err := auth.Local(db, data.Username, data.Password)
	if err != nil {
		c.AbortWithError(500, err)
		return
	}

	if errData != nil {
		c.JSON(401, errData)
		return
	}

	if usr.Administrator != "super" {
		c.JSON(401, &errortypes.ErrorData{
			Error:   "unauthorized",
			Message: "Not authorized",
		})
		return
	}

	cook := cookie.New(c.Writer, c.Request)

	_, err = cook.NewSession(db, usr.Id, true)
	if err != nil {
		c.AbortWithError(500, err)
		return
	}

	c.Status(200)
}

func logoutGet(c *gin.Context) {
	db := c.MustGet("db").(*database.Database)
	sess := c.MustGet("session").(*session.Session)

	if sess != nil {
		err := sess.Remove(db)
		if err != nil {
			c.AbortWithError(500, err)
			return
		}
	}

	c.Redirect(302, "/login")
}

func authRequestGet(c *gin.Context) {
	auth.Request(c)
}

func authCallbackGet(c *gin.Context) {
	db := c.MustGet("db").(*database.Database)
	sig := c.Query("sig")
	query := strings.Split(c.Request.URL.RawQuery, "&sig=")[0]

	usr, errData, err := auth.Local(db, sig, query)
	if err != nil {
		c.AbortWithError(500, err)
		return
	}

	if errData != nil {
		c.JSON(401, errData)
		return
	}

	if usr.Disabled || usr.Administrator != "super" {
		c.JSON(401, &errortypes.ErrorData{
			Error:   "unauthorized",
			Message: "Not authorized",
		})
		return
	}

	cook := cookie.New(c.Writer, c.Request)

	_, err = cook.NewSession(db, usr.Id, true)
	if err != nil {
		c.AbortWithError(500, err)
		return
	}

	c.Redirect(302, "/")
}
