package handlers

import (
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

func setCookie(c *gin.Context, name, value string, maxAge int, httpOnly bool) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(name, value, maxAge, "/", "", isSecureCookie(), httpOnly)
}

func clearCookie(c *gin.Context, name string) {
	setCookie(c, name, "", -1, true)
}

func isSecureCookie() bool {
	frontendURL := os.Getenv("FRONTEND_URL")
	return strings.HasPrefix(strings.ToLower(frontendURL), "https://")
}
