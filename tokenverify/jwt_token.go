package tokenverify

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/openimsdk/tools/errs"
)

const HoursOneDay = 24
const secondBefore = 5

type Claims struct {
	UserID     string
	PlatformID int // login platform
	jwt.RegisteredClaims
}

func BuildClaims(uid string, platformID int, ttl int64) Claims {
	now := time.Now()
	before := now.Add(-time.Second * time.Duration(secondBefore))
	return Claims{
		UserID:     uid,
		PlatformID: platformID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(ttl*HoursOneDay) * time.Hour)), // Expiration time
			IssuedAt:  jwt.NewNumericDate(before),                                              // Issuing time
			//NotBefore: jwt.NewNumericDate(before),                                              // Begin Effective time
			ID: uuid.New().String(),
		},
	}
}

func GetClaimFromToken(tokensString string, secretFunc jwt.Keyfunc) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokensString, &Claims{}, secretFunc)
	if err == nil {
		if claims, ok := token.Claims.(*Claims); ok && token.Valid {
			return claims, nil
		}
		return nil, errs.WrapMsg(errs.ErrTokenUnknown, "claims unknown", "token", tokensString)
	}

	switch {
	case errors.Is(err, jwt.ErrTokenMalformed):
		err = errs.ErrTokenMalformed
	case errors.Is(err, jwt.ErrTokenExpired):
		err = errs.ErrTokenExpired
	case errors.Is(err, jwt.ErrTokenNotValidYet), errors.Is(err, jwt.ErrTokenUsedBeforeIssued):
		err = errs.ErrTokenNotValidYet
	default:
		err = errs.ErrTokenUnknown
	}
	return nil, errs.WrapMsg(err, "jwt parse error", "token", tokensString)
}
