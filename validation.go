package lib

import (
	"github.com/go-playground/validator/v10"
	"gopkg.in/yaml.v3"
	"strconv"
	"strings"
)

func GetValidatorWith(validators ...string) *validator.Validate {
	result := validator.New(validator.WithRequiredStructEnabled())

	for _, name := range validators {
		if name == "min_length" {
			result.RegisterValidation("min_length", ValidateMinimumLength)
		} else if name == "max_length" {
			result.RegisterValidation("max_length", ValidateMaximumLength)
		} else if name == "string_contain" {
			result.RegisterValidation("string_contain", ValidateStringContain)
		} else if name == "string_not_contain" {
			result.RegisterValidation("string_not_contain", ValidateStringNotContain)
		} else if name == "string_start_with" {
			result.RegisterValidation("string_start_with", ValidateStringStartsWith)
		} else if name == "string_not_start_with" {
			result.RegisterValidation("string_not_start_with", ValidateStringNotStartsWith)
		} else if name == "string_end_with" {
			result.RegisterValidation("string_end_with", ValidateStringEndsWith)
		} else if name == "string_not_end_with" {
			result.RegisterValidation("string_not_end_with", ValidateStringNotEndsWith)
		} else if name == "yaml" {
			result.RegisterValidation("yaml", ValidateIsYamlFormatted)
		} else {
			panic("The validation '" + name + "' is not supported")
		}
	}

	return result
}

func ValidateStringContain(fl validator.FieldLevel) bool {
	param := fl.Param()
	value := fl.Field().String()

	return strings.Contains(value, param)
}

func ValidateStringNotContain(fl validator.FieldLevel) bool {
	param := fl.Param()
	value := fl.Field().String()

	return !strings.Contains(value, param)
}

func ValidateStringStartsWith(fl validator.FieldLevel) bool {
	param := fl.Param()
	value := fl.Field().String()

	return strings.HasPrefix(value, param)
}

func ValidateStringNotStartsWith(fl validator.FieldLevel) bool {
	param := fl.Param()
	value := fl.Field().String()

	return !strings.HasPrefix(value, param)
}

func ValidateStringEndsWith(fl validator.FieldLevel) bool {
	param := fl.Param()
	value := fl.Field().String()

	return strings.HasSuffix(value, param)
}

func ValidateStringNotEndsWith(fl validator.FieldLevel) bool {
	param := fl.Param()
	value := fl.Field().String()

	return !strings.HasSuffix(value, param)
}

func ValidateIsYamlFormatted(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	result := make(map[interface{}]interface{})

	err := yaml.Unmarshal([]byte(value), &result)
	return err == nil
}

func ValidateMinimumLength(fl validator.FieldLevel) bool {
	length, err := strconv.Atoi(fl.Param())
	if err != nil {
		return false
	}

	actualLength := getStringOrSliceLength(fl)

	return actualLength >= length
}

func ValidateMaximumLength(fl validator.FieldLevel) bool {
	length, err := strconv.Atoi(fl.Param())
	if err != nil {
		return false
	}

	actualLength := getStringOrSliceLength(fl)

	return actualLength <= length
}

func getStringOrSliceLength(fl validator.FieldLevel) int {
	value := fl.Field().Interface()
	actualLength := -1
	if strValue, ok := value.(string); ok {
		actualLength = len(strValue)
	} else if strPointerValue, ok := value.(*string); ok {
		actualLength = len(*strPointerValue)
	} else {
		// This is a slice
		actualLength = fl.Field().Len()
	}
	return actualLength
}
