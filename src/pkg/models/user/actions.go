package user

import (
	"context"
	"time"

	"api.lnlink.net/src/pkg/errs"
	"api.lnlink.net/src/pkg/global"
	"api.lnlink.net/src/pkg/models/jwt"
	"api.lnlink.net/src/pkg/services/stripe"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"
)

// creates a user, nothing to do with the auth
func CreateUser(userAuth *UserAuth) User {
	hash, err := bcrypt.GenerateFromPassword([]byte(userAuth.Password), bcrypt.DefaultCost)
	errs.Invariant(err == nil, "can't hash password")
	stripe_customer_id, err := stripe.CreateCustomer(userAuth.Email)
	errs.Invariant(err == nil, "can't create stripe customer")

	user := User{
		Email:            userAuth.Email,
		PasswordHash:     string(hash),
		ActiveTokens:     []jwt.Token{},
		StripeCustomerID: stripe_customer_id,
		TokensAvailable:  10,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	collection := global.MONGO_CLIENT.Database(global.MONGO_DB_NAME).Collection(UserCollection)
	result, err := collection.InsertOne(context.Background(), user)
	errs.Invariant(err == nil, "can't create user", err)

	user.ID = result.InsertedID.(primitive.ObjectID)

	return user
}

// get a user by their ID
func GetUserByID(userID primitive.ObjectID) *User {
	collection := global.MONGO_CLIENT.Database(global.MONGO_DB_NAME).Collection(UserCollection)

	var user User
	err := collection.FindOne(context.Background(), bson.M{"_id": userID}).Decode(&user)
	if err != nil {
		return nil
	}

	return &user
}

func GetUserByStripeCustomerID(stripeCustomerID string) *User {
	collection := global.MONGO_CLIENT.Database(global.MONGO_DB_NAME).Collection(UserCollection)

	var user User
	err := collection.FindOne(context.Background(), bson.M{"stripeCustomerID": stripeCustomerID}).Decode(&user)
	if err != nil {
		return nil
	}

	return &user
}

func (user *User) AddTokens(tokens int) {
	user = GetUserByID(user.ID)

	user.TokensAvailable += tokens

	collection := global.MONGO_CLIENT.Database(global.MONGO_DB_NAME).Collection(UserCollection)
	_, err := collection.UpdateOne(context.Background(), bson.M{"_id": user.ID}, bson.M{"$set": bson.M{"tokensAvailable": user.TokensAvailable}})
	errs.Invariant(err == nil, "can't update user")
}

// only check password, no JWT
func AuthenticateUser(userAuth *UserAuth) (bool, *User) {
	collection := global.MONGO_CLIENT.Database(global.MONGO_DB_NAME).Collection(UserCollection)

	var user User
	err := collection.FindOne(context.Background(), bson.M{"email": userAuth.Email}).Decode(&user)
	if err != nil {
		return false, nil
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(userAuth.Password))
	if err != nil {
		return false, nil
	}

	return true, &user
}

func (user *User) AddActiveToken(jwt *jwt.Token) {
	user = GetUserByID(user.ID)

	user.ActiveTokens = append(user.ActiveTokens, *jwt)

	collection := global.MONGO_CLIENT.Database(global.MONGO_DB_NAME).Collection(UserCollection)
	_, err := collection.UpdateOne(
		context.Background(),
		bson.M{"_id": user.ID},
		bson.M{"$set": bson.M{"activeTokens": user.ActiveTokens}},
	)
	errs.Invariant(err == nil, "can't update user")
}

// checks based on a string, thats why we keep them
// we automatically remove expired tokens
func (user *User) IsTokenActive(token string) bool {
	user = GetUserByID(user.ID)

	now := time.Now()
	activeTokens := make([]jwt.Token, 0)
	tokenFound := false

	for _, t := range user.ActiveTokens {
		if len(t.Value) > 0 && t.Claims.ExpiresAt > now.Unix() {
			activeTokens = append(activeTokens, t)
			if t.Value == token {
				tokenFound = true
			}
		}
	}

	collection := global.MONGO_CLIENT.Database(global.MONGO_DB_NAME).Collection(UserCollection)
	_, err := collection.UpdateOne(
		context.Background(),
		bson.M{"_id": user.ID},
		bson.M{"$set": bson.M{"activeTokens": activeTokens}},
	)
	errs.Invariant(err == nil, "can't update user")

	return tokenFound
}

// changes the password of a user
// also invalidates all active tokens
func (user *User) ChangePassword(newPassword string) {
	user = GetUserByID(user.ID)

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	errs.Invariant(err == nil, "can't hash password")

	user.PasswordHash = string(hash)
	user.UpdatedAt = time.Now()

	collection := global.MONGO_CLIENT.Database(global.MONGO_DB_NAME).Collection(UserCollection)
	_, err = collection.UpdateOne(
		context.Background(),
		bson.M{"_id": user.ID},
		bson.M{"$set": bson.M{
			"passwordHash": string(hash),
			"updatedAt":    time.Now(),
			"activeTokens": []jwt.Claims{},
		}},
	)
	errs.Invariant(err == nil, "can't update user")
}

// removes an active token from a user
// used to logout of one device
func (user *User) RemoveActiveToken(token string) {
	user = GetUserByID(user.ID)

	user.ActiveTokens = make([]jwt.Token, 0)
	for _, t := range user.ActiveTokens {
		if t.Value != token {
			user.ActiveTokens = append(user.ActiveTokens, t)
		}
	}

	collection := global.MONGO_CLIENT.Database(global.MONGO_DB_NAME).Collection(UserCollection)
	_, err := collection.UpdateOne(
		context.Background(),
		bson.M{"_id": user.ID},
		bson.M{"$set": bson.M{"activeTokens": user.ActiveTokens}},
	)
	errs.Invariant(err == nil, "can't update user")
}
