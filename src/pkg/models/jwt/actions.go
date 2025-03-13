package jwt

import (
	"time"

	"api.lnlink.net/src/pkg/global"
	jwtv5 "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func CreateJWT(userID primitive.ObjectID) (string, error) {
	now := time.Now()
	expiresAt := now.Add(DEFAULT_EXPIRATION_TIME)
	id := uuid.New().String()

	jwt := JWT{
		Issuer:    DEFAULT_ISSUER,
		Subject:   userID.Hex(),
		Audience:  DEFAULT_AUDIENCE,
		ExpiresAt: expiresAt.Unix(),
		IssuedAt:  now.Unix(),
		NotBefore: now.Unix(),
		JWTID:     id,
	}

	token := jwtv5.NewWithClaims(jwtv5.SigningMethodHS256, jwt.ToClaims())
	return token.SignedString([]byte(global.JWT_SIGNING_KEY))
}

func ValidateJWT(token string) (bool, *JWT) {
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

	jwt := &JWT{
		Issuer:    claims["iss"].(string),
		Subject:   claims["sub"].(string),
		Audience:  claims["aud"].(string),
		ExpiresAt: int64(claims["exp"].(float64)),
		IssuedAt:  int64(claims["iat"].(float64)),
		NotBefore: int64(claims["nbf"].(float64)),
		JWTID:     claims["jti"].(string),
		Token:     token,
	}

	return true, jwt
}

func (jwt *JWT) ToClaims() jwtv5.Claims {
	claims := jwtv5.MapClaims{
		"iss": jwt.Issuer,
		"sub": jwt.Subject,
		"aud": jwt.Audience,
		"exp": jwt.ExpiresAt,
		"iat": jwt.IssuedAt,
	}

	return claims
}
