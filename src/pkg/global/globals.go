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
var JWT_SIGNING_KEY = ""
var STRIPE_SECRET_KEY = ""
var TOKENS_10_ID = ""
var TOKENS_100_ID = ""
var TOKENS_1000_ID = ""
var SUCCESS_URL = ""

// mongo
var MONGO_CLIENT *mongo.Client

// Gin Router
var GIN_ROUTER *gin.Engine
