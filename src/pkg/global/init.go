package global

import (
	"context"
	"os"

	"api.lnlink.net/src/pkg/errs"

	"github.com/gin-gonic/gin"
	"github.com/stripe/stripe-go/v81"

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

	STRIPE_SECRET_KEY = os.Getenv("STRIPE_SECRET_KEY")
	errs.Invariant(len(STRIPE_SECRET_KEY) != 0, ".env file doesn't have STRIPE_SECRET_KEY")

	TOKENS_10_ID = os.Getenv("TOKENS_10_ID")
	errs.Invariant(len(TOKENS_10_ID) != 0, ".env file doesn't have TOKENS_10_ID")

	TOKENS_100_ID = os.Getenv("TOKENS_100_ID")
	errs.Invariant(len(TOKENS_100_ID) != 0, ".env file doesn't have TOKENS_100_ID")

	TOKENS_1000_ID = os.Getenv("TOKENS_1000_ID")
	errs.Invariant(len(TOKENS_1000_ID) != 0, ".env file doesn't have TOKENS_1000_ID")

	SUCCESS_URL = os.Getenv("SUCCESS_URL")
	errs.Invariant(len(SUCCESS_URL) != 0, ".env file doesn't have SUCCESS_URL")

	// connect to db
	MONGO_CLIENT, err = mongo.Connect(context.Background(), options.Client().ApplyURI(MONGO_DB_URI))
	errs.Invariant(err == nil, "can't connect to mongodb instance")

	stripe.Key = STRIPE_SECRET_KEY

	//Gin Router
	GIN_ROUTER = gin.Default()
}

func Deinit() {
	MONGO_CLIENT.Disconnect(context.Background())
}
