package global

import (
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/mongo"
)

// env variables
var MONGO_DB_URI = ""
var MONGO_DB_NAME = ""
var RESEND_API_KEY = ""
var RESEND_FROM = ""

// mongo
var MONGO_CLIENT *mongo.Client

// Gin Router
var GIN_ROUTER *gin.Engine
