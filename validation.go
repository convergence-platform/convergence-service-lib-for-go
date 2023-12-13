package lib

import (
	"github.com/go-playground/validator/v10"
	"strings"
)

func ValidateRequestDTO(requestLog *RequestLog, request any) *ManagedApiError {
	validate := validator.New()

	return ValidateRequestDTOWithCustomValidator(validate, requestLog, request)
}

func ValidateRequestDTOWithCustomValidator(validate *validator.Validate, requestLog *RequestLog, request any) *ManagedApiError {
	errs := validate.Struct(request)
	if errs != nil {
		result := &ManagedApiError{
			HttpStatusCode:  400,
			Code:            INVALID_DATA,
			Message:         "The request input is invalid, refer to body for details.",
			RequestId:       requestLog.GetRawRequestID(),
			ParentRequestId: requestLog.ParentRequestIdentifier,
		}

		body := &RequestValidationFailureDTO{
			Errors: make([]*RequestValidationFieldFailureDTO, 0),
		}
		result.SetBody(body, "request_error_info")

		fieldToErrorInfo := make(map[string]*RequestValidationFieldFailureDTO)
		for _, err := range errs.(validator.ValidationErrors) {
			var fieldInfo *RequestValidationFieldFailureDTO
			ok := false

			if fieldInfo, ok = fieldToErrorInfo[err.StructNamespace()]; !ok {
				fieldInfo = &RequestValidationFieldFailureDTO{
					Field:    toSnakeCase(err.Field()),
					Location: "body",
					Messages: make([]string, 0),
				}
				body.Errors = append(body.Errors, fieldInfo)
			}

			errorMessage := "Failing to pass validation: `" + err.Tag() + "`"
			fieldInfo.Messages = append(fieldInfo.Messages, errorMessage)
		}

		return result
	}

	return nil
}

func toSnakeCase(field string) string {
	result := make([]string, 0)

	for i := 0; i < len(field); i++ {
		ch := field[i]
		if ch >= 'A' && ch <= 'Z' {
			if i > 0 {
				result = append(result, "_")
			}
			result = append(result, strings.ToLower(field[i:i+1]))
		} else {
			result = append(result, field[i:i+1])
		}
	}

	return strings.Join(result, "")
}
