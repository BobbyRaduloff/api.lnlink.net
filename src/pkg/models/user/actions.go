package user

import (
	"context"
	"time"

	"api.lnlink.net/src/pkg/errs"
	"api.lnlink.net/src/pkg/global"
	"api.lnlink.net/src/pkg/models/jwt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"
)

func CreateUser(userAuth *UserAuth) User {
	hash, err := bcrypt.GenerateFromPassword([]byte(userAuth.Password), bcrypt.DefaultCost)
	errs.Invariant(err == nil, "can't hash password")

	user := User{
		Email:        userAuth.Email,
		PasswordHash: string(hash),
		ActiveTokens: []jwt.JWT{},
	}

	collection := global.MONGO_CLIENT.Database(global.MONGO_DB_NAME).Collection(UserCollection)
	result, err := collection.InsertOne(context.Background(), user)
	errs.Invariant(err == nil, "can't create user")

	user.ID = result.InsertedID.(primitive.ObjectID)
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()

	return user
}

func GetUserByID(userID primitive.ObjectID) *User {
	collection := global.MONGO_CLIENT.Database(global.MONGO_DB_NAME).Collection(UserCollection)

	var user User
	err := collection.FindOne(context.Background(), bson.M{"_id": userID}).Decode(&user)
	if err != nil {
		return nil
	}

	return &user
}

// only check password
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

func (user *User) AddActiveToken(token string) {
	user = GetUserByID(user.ID)

	user.ActiveTokens = append(user.ActiveTokens, jwt.JWT{
		Token:     token,
		ExpiresAt: time.Now().Add(time.Hour * 24).Unix(),
	})

	collection := global.MONGO_CLIENT.Database(global.MONGO_DB_NAME).Collection(UserCollection)
	_, err := collection.UpdateOne(
		context.Background(),
		bson.M{"_id": user.ID},
		bson.M{"$set": bson.M{"activeTokens": user.ActiveTokens}},
	)
	errs.Invariant(err == nil, "can't update user")
}

// we automatically remove expired tokens
func (user *User) IsTokenActive(token string) bool {
	user = GetUserByID(user.ID)

	now := time.Now()
	activeTokens := make([]jwt.JWT, 0)
	tokenFound := false

	for _, t := range user.ActiveTokens {
		if t.ExpiresAt > now.Unix() {
			activeTokens = append(activeTokens, t)
			if t.Token == token {
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
			"activeTokens": []jwt.JWT{},
		}},
	)
	errs.Invariant(err == nil, "can't update user")
}
