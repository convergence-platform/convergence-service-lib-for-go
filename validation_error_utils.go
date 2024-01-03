package lib

import (
	"github.com/go-playground/validator/v10"
	"strings"
)

func CreateBadRequestInvalidJSON(requestLog *RequestLog) *ManagedApiError {
	return &ManagedApiError{
		HttpStatusCode:  400,
		Code:            API_UNPARSABLE_JSON,
		Message:         "The request input is not a valid JSON.",
		RequestId:       requestLog.GetRawRequestID(),
		ParentRequestId: requestLog.ParentRequestIdentifier,
	}
}

func CreateBadRequestInvalidFieldProvided(errs error, requestLog *RequestLog) *ManagedApiError {
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
			fieldName := err.StructNamespace()
			fieldName = fieldName[strings.Index(fieldName, ".")+1:]
			fieldInfo = &RequestValidationFieldFailureDTO{
				Field:    ConvertPascalToSnake(fieldName),
				Location: "body",
				Messages: make([]string, 0),
			}
			body.Errors = append(body.Errors, fieldInfo)
		}

		errorMessage := "Failing to pass validation: '" + err.Tag() + "'"
		fieldInfo.Messages = append(fieldInfo.Messages, errorMessage)
	}

	return result
}

func ConvertPascalToSnake(pascal string) string {
	lastLowerCase := false
	result := ""

	for i := 0; i < len(pascal); i++ {
		ch := pascal[i : i+1]
		if isUpperCaseChar(ch) {
			if lastLowerCase {
				result += "_"
			}
			result += strings.ToLower(ch)
			lastLowerCase = false
		} else if ch == "." {
			result += ch
			lastLowerCase = false
		} else {
			result += ch
			lastLowerCase = true
		}
	}

	return result
}

func isUpperCaseChar(input string) bool {
	chars := "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	return strings.Contains(chars, input)
}
