package api_server

import (
	"net/http"

	"api.lnlink.net/src/pkg/models/jwt"
	"api.lnlink.net/src/pkg/models/user"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	UserIDKey = "userID"
	TokenKey  = "token"
)

func RegisterAuthRoutes(r *gin.Engine) {
	r.POST("/api/auth/login", LoginUser)
	r.PATCH("/api/auth/password", AuthMiddleware(), ChangePassword)
	r.DELETE("/api/auth/logout", AuthMiddleware(), LogoutUser)
}

func LoginUser(c *gin.Context) {
	var userAuth user.UserAuth
	if err := c.ShouldBindJSON(&userAuth); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	success, user := user.AuthenticateUser(&userAuth)
	if !success {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
		return
	}

	jwt, err := jwt.CreateJWT(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	user.AddActiveToken(&jwt)

	c.JSON(http.StatusOK, gin.H{"accessToken": jwt.Value})
}

func LogoutUser(c *gin.Context) {
	userID := GetUserID(c)
	user.GetUserByID(userID).RemoveActiveToken(GetToken(c))
	c.JSON(http.StatusOK, gin.H{"message": "Ok"})
}

func ChangePassword(c *gin.Context) {
	var userChangePassword user.UserChangePassword
	if err := c.ShouldBindJSON(&userChangePassword); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	userID := GetUserID(c)
	currentUser := user.GetUserByID(userID)
	if currentUser == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token payload"})
		c.Abort()
		return
	}

	success, currentUser := user.AuthenticateUser(&user.UserAuth{
		Email:    currentUser.Email,
		Password: userChangePassword.OldPassword,
	})
	if !success {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid old password"})
		return
	}

	currentUser.ChangePassword(userChangePassword.NewPassword)
	c.JSON(http.StatusOK, gin.H{"message": "Password changed"})
}

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		if len(authHeader) < 7 || authHeader[:7] != "Bearer " {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header format"})
			c.Abort()
			return
		}

		token := authHeader[7:]

		valid, jwtToken := jwt.ValidateJWT(token)
		if !valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		userID, err := primitive.ObjectIDFromHex(jwtToken.Claims.Subject)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token subject is not a valid object ID"})
			c.Abort()
			return
		}

		user := user.GetUserByID(userID)
		if user == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
			c.Abort()
			return
		}

		if len(user.ActiveTokens) == 0 {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User has no active tokens"})
			c.Abort()
			return
		}

		foundActiveToken := false
		for _, token := range user.ActiveTokens {
			if token.Value == jwtToken.Value {
				foundActiveToken = true
				break
			}
		}

		if !foundActiveToken {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token is not active"})
			c.Abort()
			return
		}

		c.Set(UserIDKey, userID)
		c.Set(TokenKey, token)

		c.Next()
	}
}

func GetUserID(c *gin.Context) primitive.ObjectID {
	userID, exists := c.Get(UserIDKey)
	if !exists {
		return primitive.NilObjectID
	}
	return userID.(primitive.ObjectID)
}

func GetToken(c *gin.Context) string {
	token, exists := c.Get(TokenKey)
	if !exists {
		return ""
	}
	return token.(string)
}
