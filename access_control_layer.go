package lib

import (
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

func IsSignedIn() EndpointAuthorizationHandler {
	return func(context *fiber.Ctx, token *jwt.Token, hadAuthorizationHeader bool) *string {
		if token != nil && hadAuthorizationHeader {
			return nil
		}

		message := ""
		return &message
	}
}

func IsNotSignedIn() EndpointAuthorizationHandler {
	return func(context *fiber.Ctx, token *jwt.Token, hadAuthorizationHeader bool) *string {
		if token == nil && !hadAuthorizationHeader {
			return nil
		}

		message := ""

		if hadAuthorizationHeader {
			message = "Endpoint is available for anonymous users only."
		}

		return &message
	}
}

func AllowAll() EndpointAuthorizationHandler {
	return func(context *fiber.Ctx, token *jwt.Token, hadAuthorizationHeader bool) *string {
		return nil
	}
}

func IsServiceCall() EndpointAuthorizationHandler {
	return func(context *fiber.Ctx, token *jwt.Token, hadAuthorizationHeader bool) *string {
		message := ""
		if !hadAuthorizationHeader || token == nil || token.Claims == nil {
			return &message
		}

		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			if isService, exists := claims["is_inter_service_call"]; exists {
				if result, typeMatch := isService.(bool); typeMatch {
					if result {
						return nil
					} else {
						return &message
					}
				}
			}
		}

		return &message
	}
}

func HasAuthority(authority string) EndpointAuthorizationHandler {
	return func(context *fiber.Ctx, token *jwt.Token, hadAuthorizationHeader bool) *string {
		message := ""
		if !hadAuthorizationHeader || token == nil || token.Claims == nil {
			return &message
		}

		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			authorities := claims["authorities"]
			if authorities != nil {
				for _, authorityInJwt := range authorities.([]interface{}) {
					if authorityAsString, ok := authorityInJwt.(string); ok {
						if authorityAsString == authority {
							return nil
						}
					}
				}
			}
		}

		return &message
	}
}
