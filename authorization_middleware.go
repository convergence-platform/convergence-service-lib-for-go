package lib

import (
	"crypto/ecdsa"
	"encoding/json"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"strings"
)

type AuthorizationMiddlewareConfig struct {
	PublicKey *ecdsa.PublicKey
}

var authorizationConfig *AuthorizationMiddlewareConfig

func AuthorizationMiddleware(context *fiber.Ctx) error {
	if authorizationConfig == nil {
		signingKey := ServiceInstance.GetConfiguration("security.authentication.secret").(string)
		signingKey = strings.Replace(signingKey, "\\n", "\n", -1)

		authorizationConfig = &AuthorizationMiddlewareConfig{
			PublicKey: &DecodePrivate(signingKey).PublicKey,
		}
	}

	path := context.OriginalURL()
	method := context.Method("")

	endpointInfo, pathMatched := getEndpointInfo(path, method)
	if endpointInfo == nil {
		if pathMatched {
			return notAllowedMethodResponse(context)
		} else {
			return notFoundResponse(context)
		}
	} else {
		authorizationHeader := getAuthorizationHeader(context)
		var token *jwt.Token
		var managedError *ManagedApiError
		if authorizationHeader != nil {
			token, managedError = isValidAuthorizationToken(*authorizationHeader)
			if managedError != nil {
				return convertManagedApiErrorToResponse(context, managedError)
			} else if token != nil && !token.Valid {
				bodyType := "api_failure"
				managedError = &ManagedApiError{
					Code:     API_INTERNAL_ERROR,
					Message:  "Authorization token validation failed for unknown reason",
					body:     nil,
					bodyType: &bodyType,
				}

				return convertManagedApiErrorToResponse(context, managedError)
			}
		}

		validAuthorization, customErrorMessage := isAuthorized(endpointInfo, context, token, authorizationHeader != nil)
		if endpointInfo.Authorization == nil || validAuthorization {
			return context.Next()
		} else {
			return unauthorizedResponseInvalidToken(context, customErrorMessage)
		}
	}
}

func getEndpointInfo(url string, method string) (*ServiceEndpointAuthorizationDetails, bool) {
	var result *ServiceEndpointAuthorizationDetails
	pathMatched := false
	method = strings.ToUpper(method)

	for _, info := range ServiceInstance.endpointsAuthorization {
		urlMatched := matchURLToEndpoint(url, info.URL)
		if urlMatched {
			pathMatched = true
		}
		if info.Method == method && urlMatched {
			result = info
			break
		}
	}

	return result, pathMatched
}

func matchURLToEndpoint(url string, epUrl string) bool {
	if url == epUrl {
		return true
	}
	compsURL := strings.Split(url, "/")
	compsEpURL := strings.Split(epUrl, "/")

	if len(compsURL) == len(compsEpURL) {
		for i, a := range compsURL {
			b := compsEpURL[i]
			if (strings.HasPrefix(b, "{") && strings.HasSuffix(b, "}")) || a == b {
				continue
			}
			return false
		}

		return true
	} else {
		return false
	}
}

func isAuthorized(info *ServiceEndpointAuthorizationDetails, context *fiber.Ctx, token *jwt.Token, hadAuthorizationHeader bool) (bool, *string) {
	if info.Authorization == nil {
		return true, nil
	}

	result := info.Authorization(context, token, hadAuthorizationHeader)
	return result == nil, result
}

func convertManagedApiErrorToResponse(context *fiber.Ctx, err *ManagedApiError) error {
	requestLog := InitializeRequestLogForGatewayMiddleware(context)

	statusCode := err.HttpStatusCode
	context.Status(statusCode)
	bodyType := err.bodyType

	response := ApiResponse[any]{
		Header: ResponseHeaderDTO{
			BodyType:        bodyType,
			HttpStatusCode:  statusCode,
			Code:            err.Code,
			Message:         err.Message,
			RequestId:       requestLog.GetRawRequestID(),
			ParentRequestId: requestLog.ParentRequestIdentifier,
		},
		Body: nil,
	}

	FinishRequestLog(requestLog, &response)

	jsonString, _ := json.Marshal(response)

	context.Set("Content-Type", "application/json")
	return context.SendString(string(jsonString))
}

func getAuthorizationHeader(context *fiber.Ctx) *string {
	authorizationHeaders := context.GetReqHeaders()["Authorization"]
	if authorizationHeaders == nil {
		return nil
	}

	return &authorizationHeaders[0]
}

func isValidAuthorizationToken(authHeader string) (*jwt.Token, *ManagedApiError) {
	bearerLength := len("Bearer ")
	bodyType := "api_failure"
	if !strings.HasPrefix(authHeader, "Bearer ") && !strings.HasPrefix(authHeader, "API-Key ") {
		return nil, &ManagedApiError{
			HttpStatusCode: 401,
			Code:           INVALID_AUTHORIZATION_TOKEN,
			Message:        "Service expects an authorization token that starts wither with Bearer or API-Key.",
			body:           nil,
			bodyType:       &bodyType,
		}
	} else if strings.HasPrefix(authHeader, "Bearer ") {
		token := authHeader[bearerLength:]
		payload, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
			return authorizationConfig.PublicKey, nil
		})

		if err == nil {
			return payload, nil
		}

		if err.Error() == "token has invalid claims: token is expired" {
			return nil, &ManagedApiError{
				HttpStatusCode: 401,
				Code:           EXPIRED_AUTHORIZATION_TOKEN,
				Message:        "Authorization token is expired, please refresh the token or get a new one.",
				body:           nil,
				bodyType:       &bodyType,
			}
		}

		if err.Error() == "token signature is invalid: crypto/ecdsa: verification error" {
			return nil, &ManagedApiError{
				HttpStatusCode: 403,
				Code:           INVALID_AUTHORIZATION_TOKEN,
				Message:        "Authorization token is invalid, this incident will be reported.",
				body:           nil,
				bodyType:       &bodyType,
			}
		}
	} else if strings.HasPrefix(authHeader, "API-Key ") {
		return nil, nil
	}

	return nil, &ManagedApiError{
		HttpStatusCode: 403,
		Code:           API_INTERNAL_ERROR,
		Message:        "Authorization token verification failed due to unknown error, likely an invalid token.",
		body:           nil,
		bodyType:       &bodyType,
	}
}

func unauthorizedResponseInvalidToken(context *fiber.Ctx, message *string) error {
	requestLog := InitializeRequestLogForGatewayMiddleware(context)

	statusCode := 403
	context.Status(statusCode)
	bodyType := "failure_info"

	errorMessage := ""
	if message != nil {
		errorMessage = *message
	} else {
		errorMessage = "The authorization token is invalid for path " + context.OriginalURL()
	}

	response := ApiResponse[any]{
		Header: ResponseHeaderDTO{
			BodyType:        &bodyType,
			HttpStatusCode:  statusCode,
			Code:            INVALID_AUTHORIZATION_TOKEN,
			Message:         errorMessage,
			RequestId:       requestLog.GetRawRequestID(),
			ParentRequestId: requestLog.ParentRequestIdentifier,
		},
		Body: nil,
	}

	FinishRequestLog(requestLog, &response)

	jsonString, _ := json.Marshal(response)

	context.Set("Content-Type", "application/json")
	return context.SendString(string(jsonString))
}

func notFoundResponse(context *fiber.Ctx) error {
	requestLog := InitializeRequestLogForGatewayMiddleware(context)

	statusCode := 404
	context.Status(statusCode)
	bodyType := "failure_info"

	response := ApiResponse[any]{
		Header: ResponseHeaderDTO{
			BodyType:        &bodyType,
			HttpStatusCode:  statusCode,
			Code:            API_RESOURCE_NOT_FOUND,
			Message:         "Unable to find resource at path " + context.OriginalURL(),
			RequestId:       requestLog.GetRawRequestID(),
			ParentRequestId: requestLog.ParentRequestIdentifier,
		},
		Body: nil,
	}

	FinishRequestLog(requestLog, &response)

	jsonString, _ := json.Marshal(response)

	context.Set("Content-Type", "application/json")
	return context.SendString(string(jsonString))
}

func notAllowedMethodResponse(context *fiber.Ctx) error {
	requestLog := InitializeRequestLogForGatewayMiddleware(context)

	statusCode := 405
	context.Status(statusCode)
	bodyType := "failure_info"
	method := context.Method("")

	response := ApiResponse[any]{
		Header: ResponseHeaderDTO{
			BodyType:        &bodyType,
			HttpStatusCode:  statusCode,
			Code:            API_METHOD_NOT_ALLOWED,
			Message:         "Unable to find resource at path " + method + " " + context.OriginalURL(),
			RequestId:       requestLog.GetRawRequestID(),
			ParentRequestId: requestLog.ParentRequestIdentifier,
		},
		Body: nil,
	}

	FinishRequestLog(requestLog, &response)

	jsonString, _ := json.Marshal(response)

	context.Set("Content-Type", "application/json")
	return context.SendString(string(jsonString))
}
