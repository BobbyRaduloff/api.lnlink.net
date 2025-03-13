package user

import (
	"time"

	"api.lnlink.net/src/pkg/models/jwt"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var UserCollection = "users"

// we keep track of active tokens to prevent token reuse
// for example, if a user changes their password, we can invalidate all their active tokens
// TODO: implement login persistance via refresh tokens i.e. we issue a refresh token as a http-only cookie
// and we use it to issue new access tokens upon expiration.
// TODO: implement a registration flow i.e. we need to verify the email address
type User struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	Email        string             `bson:"email" json:"email"`
	PasswordHash string             `bson:"passwordHash" json:"passwordHash"`
	ActiveTokens []jwt.JWT          `bson:"activeTokens" json:"activeTokens"`

	CreatedAt time.Time `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time `bson:"updatedAt" json:"updatedAt"`
}

type UserAuth struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type UserChangePassword struct {
	OldPassword string `json:"oldPassword"`
	NewPassword string `json:"newPassword"`
}
