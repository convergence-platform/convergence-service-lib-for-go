package lib

import (
	"github.com/go-playground/validator/v10"
	"gopkg.in/yaml.v3"
	"strconv"
	"strings"
)

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
