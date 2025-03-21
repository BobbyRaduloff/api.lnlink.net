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
var RUNPOD_API_KEY = ""
var S3_REGION = ""
var S3_ACCESS_KEY_ID = ""
var S3_SECRET_ACCESS_KEY = ""
var S3_INPUT_BUCKET_NAME = ""
var S3_OUTPUT_BUCKET_NAME = ""
var S3_MODEL_BUCKET_NAME = ""

// mongo
var MONGO_CLIENT *mongo.Client

// Gin Router
var GIN_ROUTER *gin.Engine
