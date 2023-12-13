package lib

import (
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

func IsSignedIn() EndpointAuthorizationHandler {
	return func(context *fiber.Ctx, token *jwt.Token) bool {
		return token != nil
	}
}

func HasAuthority(authority string) EndpointAuthorizationHandler {
	return func(context *fiber.Ctx, token *jwt.Token) bool {
		if token == nil || token.Claims == nil {
			return false
		}

		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			authorities := claims["authorities"]
			if authorities != nil {
				for _, authorityInJwt := range authorities.([]interface{}) {
					if authorityAsString, ok := authorityInJwt.(string); ok {
						if authorityAsString == authority {
							return true
						}
					}
				}
			}
		}

		return false
	}
}
