package jwt

import (
	"time"

	"api.lnlink.net/src/pkg/global"
	jwtv5 "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// creates a JWT for a user ID
func CreateJWT(userID primitive.ObjectID) (Token, error) {
	now := time.Now()
	expiresAt := now.Add(DEFAULT_EXPIRATION_TIME)
	id := uuid.New().String()

	claims := Claims{
		Issuer:    DEFAULT_ISSUER,
		Subject:   userID.Hex(),
		Audience:  DEFAULT_AUDIENCE,
		ExpiresAt: expiresAt.Unix(),
		IssuedAt:  now.Unix(),
		NotBefore: now.Unix(),
		JWTID:     id,
	}

	token := jwtv5.NewWithClaims(jwtv5.SigningMethodHS256, claims.ToRawClaims())
	tokenString, err := token.SignedString([]byte(global.JWT_SIGNING_KEY))
	if err != nil {
		return Token{}, err
	}

	return Token{
		Claims: claims,
		Value:  tokenString,
	}, nil
}

// validate's a token string and returns the token and the claims
func ValidateJWT(token string) (bool, *Token) {
	parsedToken, err := jwtv5.Parse(token, func(token *jwtv5.Token) (interface{}, error) {
		return []byte(global.JWT_SIGNING_KEY), nil
	}, jwtv5.WithValidMethods([]string{jwtv5.SigningMethodHS256.Name}))

	if err != nil || !parsedToken.Valid {
		return false, nil
	}

	claims, ok := parsedToken.Claims.(jwtv5.MapClaims)
	if !ok {
		return false, nil
	}

	parsedClaims := Claims{
		Issuer:    claims["iss"].(string),
		Subject:   claims["sub"].(string),
		Audience:  claims["aud"].(string),
		ExpiresAt: int64(claims["exp"].(float64)),
		IssuedAt:  int64(claims["iat"].(float64)),
		NotBefore: int64(claims["nbf"].(float64)),
		JWTID:     claims["jti"].(string),
	}

	if parsedClaims.ExpiresAt < time.Now().Unix() {
		return false, nil
	}

	return true, &Token{
		Claims: parsedClaims,
		Value:  token,
	}
}

// helper to convert the claims to a raw jwtv5.Claims
func (jwt *Claims) ToRawClaims() jwtv5.Claims {
	claims := jwtv5.MapClaims{
		"iss": jwt.Issuer,
		"sub": jwt.Subject,
		"aud": jwt.Audience,
		"exp": jwt.ExpiresAt,
		"iat": jwt.IssuedAt,
		"nbf": jwt.NotBefore,
		"jti": jwt.JWTID,
	}

	return claims
}
