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
)

func RegisterAuthRoutes(r *gin.Engine) {
	r.POST("/api/auth/login", LoginUser)
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

	user.AddActiveToken(jwt)

	c.JSON(http.StatusOK, gin.H{"accessToken": jwt})
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

		userID, err := primitive.ObjectIDFromHex(jwtToken.Subject)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token payload"})
			c.Abort()
			return
		}

		user := user.GetUserByID(userID)
		if user == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token payload"})
			c.Abort()
			return
		}

		if len(user.ActiveTokens) == 0 {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token payload"})
			c.Abort()
			return
		}

		for _, token := range user.ActiveTokens {
			if token.Token == jwtToken.Token {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token payload"})
				c.Abort()
				return
			}
		}

		c.Set(UserIDKey, userID)

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
