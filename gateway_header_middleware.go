package lib

import (
	"encoding/json"
	"github.com/gofiber/fiber/v2"
	uuid2 "github.com/google/uuid"
	"slices"
	"strings"
)

type GatewayHeaderValidationMiddlewareConfig struct {
	IsBehindGateway                 bool
	ReservedMandatoryGatewayHeaders []string
	ReservedOptionalGatewayHeaders  []string
	RequestType                     string
}

var gatewayConfig *GatewayHeaderValidationMiddlewareConfig

func GatewayHeaderValidationMiddleware(context *fiber.Ctx) error {
	if gatewayConfig == nil {
		gatewayConfig = &GatewayHeaderValidationMiddlewareConfig{
			IsBehindGateway: ServiceInstance.GetBooleanConfiguration("security.is_behind_gateway"),
			ReservedMandatoryGatewayHeaders: []string{
				REQUEST_ID_HEADER,
				CALLER_SERVICE_HEADER,
				CALLER_SERVICE_HASH_HEADER,
				CALLER_SERVICE_VERSION_HEADER,
			},
			ReservedOptionalGatewayHeaders: []string{
				PARENT_REQUEST_ID_HEADER,
			},
			RequestType: ServiceInstance.GetConfiguration("observability.request_id_prefix").(string),
		}
	}

	if gatewayConfig.IsBehindGateway {
		// Behind gateway, no need to validate token
		isMissing, missingHeaders := isMissingAnyOfGatewayHeaders(context)
		if isMissing {
			return requestIsMissingGatewayHeaders(context, missingHeaders)
		} else {
			return context.Next()
		}
	} else if len(getIndependentServiceRequestInvalidHeaders(context)) > 0 {
		// Independent service, but contains reserved headers, must fail
		return requestHasReservedHeadersResponse(context)
	} else if requestHasAuthorizationHeader(context) {
		// Independent service, has authorization, so must be validated
		if isAuthorizationHeaderValid(context) {
			return context.Next()
		} else {
			return unauthorizedResponseCantValidateToken(context)
		}
	} else {
		// Independent service, no authorization, can proceed. If the endpoint need authorization, it will be
		// dropped by authorization filter
		return context.Next()
	}
}

func requestIsMissingGatewayHeaders(context *fiber.Ctx, missingHeaders []string) error {
	requestLog := InitializeRequestLogForGatewayMiddleware(context)

	statusCode := 502
	context.Status(statusCode)
	bodyType := "request_error_info"

	errors := make([]*RequestValidationFieldFailureDTO, 0)

	for _, header := range missingHeaders {
		details := &RequestValidationFieldFailureDTO{
			Field:    header,
			Location: "header",
			Messages: []string{"Request is missing header."},
		}
		errors = append(errors, details)
	}

	body := RequestValidationFailureDTO{
		Errors: errors,
	}

	response := ApiResponse[RequestValidationFailureDTO]{
		Header: ResponseHeaderDTO{
			BodyType:        &bodyType,
			HttpStatusCode:  statusCode,
			Code:            INVALID_DATA,
			Message:         "The request input was invalid, missing the mandatory API gateway headers.",
			RequestId:       requestLog.GetRawRequestID(),
			ParentRequestId: requestLog.ParentRequestIdentifier,
		},
		Body: body,
	}

	FinishRequestLog(requestLog, &response)

	jsonString, _ := json.Marshal(response)

	context.Set("Content-Type", "application/json")
	return context.SendString(string(jsonString))
}

func isMissingAnyOfGatewayHeaders(context *fiber.Ctx) (bool, []string) {
	headersMap := context.GetReqHeaders()
	headersList := make([]string, 0)

	for k, _ := range headersMap {
		key := strings.ToUpper(k)
		if !slices.Contains(headersList, key) {
			headersList = append(headersList, key)
		}
	}

	missingHeaders := make([]string, 0)
	for _, header := range gatewayConfig.ReservedMandatoryGatewayHeaders {
		h := strings.ToUpper(header)
		if !slices.Contains(headersList, h) {
			missingHeaders = append(missingHeaders, h)
		}
	}

	return len(missingHeaders) > 0, missingHeaders
}

func getIndependentServiceRequestInvalidHeaders(context *fiber.Ctx) []string {
	headersMap := context.GetReqHeaders()
	headersList := make([]string, 0)

	for k, _ := range headersMap {
		if !slices.Contains(headersList, k) {
			headersList = append(headersList, strings.ToUpper(k))
		}
	}

	invalidHeaders := make([]string, 0)
	for _, header := range gatewayConfig.ReservedMandatoryGatewayHeaders {
		if slices.Contains(headersList, header) {
			invalidHeaders = append(invalidHeaders, header)
		}
	}

	return invalidHeaders
}

func requestHasAuthorizationHeader(context *fiber.Ctx) bool {
	headersMap := context.GetReqHeaders()

	for k, _ := range headersMap {
		if strings.ToLower(k) == "authorization" {
			return true
		}
	}

	return false
}

func isAuthorizationHeaderValid(context *fiber.Ctx) bool {
	// TODO: implement
	return true
}

func unauthorizedResponseCantValidateToken(context *fiber.Ctx) error {
	requestLog := InitializeRequestLogForGatewayMiddleware(context)

	statusCode := 403
	context.Status(statusCode)
	bodyType := "failure_info"

	response := ApiResponse[any]{
		Header: ResponseHeaderDTO{
			BodyType:        &bodyType,
			HttpStatusCode:  statusCode,
			Code:            INVALID_AUTHORIZATION_TOKEN,
			Message:         "Unable to verify the validity of the Authorization token provided.",
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

func requestHasReservedHeadersResponse(context *fiber.Ctx) error {
	requestLog := InitializeRequestLogForGatewayMiddleware(context)
	invalidHeaders := getIndependentServiceRequestInvalidHeaders(context)

	statusCode := 400
	context.Status(statusCode)
	bodyType := "request_error_info"

	errors := make([]*RequestValidationFieldFailureDTO, 0)

	for _, header := range invalidHeaders {
		details := &RequestValidationFieldFailureDTO{
			Field:    header,
			Location: "header",
			Messages: []string{"Request includes reserved header."},
		}
		errors = append(errors, details)
	}

	body := RequestValidationFailureDTO{
		Errors: errors,
	}

	response := ApiResponse[RequestValidationFailureDTO]{
		Header: ResponseHeaderDTO{
			BodyType:        &bodyType,
			HttpStatusCode:  statusCode,
			Code:            INVALID_DATA,
			Message:         "The request input was invalid, including reserved API gateway headers.",
			RequestId:       getRequestIdFromHeader(context), // This is not returned in the request log
			ParentRequestId: requestLog.ParentRequestIdentifier,
		},
		Body: body,
	}

	FinishRequestLog(requestLog, &response)

	jsonString, _ := json.Marshal(response)

	context.Set("Content-Type", "application/json")
	return context.SendString(string(jsonString))
}

func getRequestIdFromHeader(context *fiber.Ctx) *uuid2.UUID {
	headers := context.GetReqHeaders()

	for k, v := range headers {
		if strings.ToUpper(k) == strings.ToUpper(REQUEST_ID_HEADER) {
			uuid, _ := uuid2.Parse(v[0])
			return &uuid
		}
	}

	return nil
}
