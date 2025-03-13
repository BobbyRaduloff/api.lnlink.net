package global

import (
	"context"
	"os"

	"api.lnlink.net/src/pkg/errs"

	"github.com/gin-gonic/gin"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func Init() {
	// load env
	err := godotenv.Load()
	errs.Invariant(err == nil, "no .env file found: ")

	// check for env variable correctness
	MONGO_DB_URI = os.Getenv("MONGO_DB_URI")
	errs.Invariant(len(MONGO_DB_URI) != 0, ".env file doesn't have MONGO_DB_URI")

	MONGO_DB_NAME = os.Getenv("MONGO_DB_NAME")
	errs.Invariant(len(MONGO_DB_NAME) != 0, ".env file doesn't have MONGO_DB_NAME")

	RESEND_FROM = os.Getenv("RESEND_FROM")
	errs.Invariant(len(RESEND_FROM) != 0, ".env file doesn't have RESEND_FROM")

	RESEND_API_KEY = os.Getenv("RESEND_API_KEY")
	errs.Invariant(len(RESEND_API_KEY) != 0, ".env file doesn't have RESEND_API_KEY")

	JWT_SIGNING_KEY = os.Getenv("JWT_SIGNING_KEY")
	errs.Invariant(len(JWT_SIGNING_KEY) != 0, ".env file doesn't have JWT_SIGNING_KEY")

	// connect to db
	MONGO_CLIENT, err = mongo.Connect(context.Background(), options.Client().ApplyURI(MONGO_DB_URI))
	errs.Invariant(err == nil, "can't connect to mongodb instance")

	//Gin Router
	GIN_ROUTER = gin.Default()
}

func Deinit() {
	MONGO_CLIENT.Disconnect(context.Background())
}
