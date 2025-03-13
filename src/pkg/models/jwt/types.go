package jwt

import "time"

// this is the payload of the JWT
type Claims struct {
	Issuer    string `bson:"iss" json:"iss"`
	Subject   string `bson:"sub" json:"sub"`
	Audience  string `bson:"aud" json:"aud"`
	ExpiresAt int64  `bson:"exp" json:"exp"`
	IssuedAt  int64  `bson:"iat" json:"iat"`
	NotBefore int64  `bson:"nbf" json:"nbf"`
	JWTID     string `bson:"jti" json:"jti"`
}

// its useful to store the value and the decoded claims
type Token struct {
	Claims Claims
	Value  string
}

var DEFAULT_ISSUER = "api.lnlink.net"
var DEFAULT_EXPIRATION_TIME = 12 * time.Hour
var DEFAULT_AUDIENCE = "lnlink.net"
