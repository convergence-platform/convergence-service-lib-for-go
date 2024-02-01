package lib

import (
	"bytes"
	"encoding/json"
	"github.com/golang-jwt/jwt/v5"
	uuid2 "github.com/google/uuid"
	"io"
	"net/http"
	"time"
)

const (
	SERVICE_INTERNAL string = "internal"
	SERVICE_EXTERNAL        = "external"
)

type DTOCreator func() interface{}

type BaseServiceClient struct {
	SigningKey         string
	Mode               string
	URL                string
	VerifySSL          bool
	TypeMapper         map[string]DTOCreator
	CallerServiceName  string
	ServiceName        string
	ServiceVersionHash string
	ServiceVersion     string
}

func (c BaseServiceClient) GetServiceURL() string {
	if c.URL == "" {
		panic("not implemented")
	}

	return c.URL
}

func MakeGetCall(client *BaseServiceClient, requestLog *RequestLog, url string, requiredAuthorization string) *ApiResponse[any] {
	var result *ApiResponse[any]

	targetUrl := client.GetServiceURL() + url
	request, _ := http.NewRequest("GET", targetUrl, nil)

	request.Header.Set("Content-Type", "application/json")
	fillAuthorizationHeader(client, requiredAuthorization, request)
	fillRequestIdHeaders(client, requestLog, request)

	httpClient := &http.Client{}
	response, err := httpClient.Do(request)
	if err != nil {
		result = buildFailureInfo(err)
	} else {
		data, _ := io.ReadAll(response.Body)
		result = buildResponse(data, client, response)
	}

	return result
}

func MakePostCall(client *BaseServiceClient, requestLog *RequestLog, url string, requiredAuthorization string, requestBody any) *ApiResponse[any] {
	return MakeVerbCall("POST", client, requestLog, url, requiredAuthorization, requestBody)
}

func MakePatchCall(client *BaseServiceClient, requestLog *RequestLog, url string, requiredAuthorization string, requestBody any) *ApiResponse[any] {
	return MakeVerbCall("PATCH", client, requestLog, url, requiredAuthorization, requestBody)
}

func MakePutCall(client *BaseServiceClient, requestLog *RequestLog, url string, requiredAuthorization string, requestBody any) *ApiResponse[any] {
	return MakeVerbCall("PUT", client, requestLog, url, requiredAuthorization, requestBody)
}

func MakeDeleteCall(client *BaseServiceClient, requestLog *RequestLog, url string, requiredAuthorization string, requestBody any) *ApiResponse[any] {
	return MakeVerbCall("DELETE", client, requestLog, url, requiredAuthorization, requestBody)
}

func MakeVerbCall(verb string, client *BaseServiceClient, requestLog *RequestLog, url string, requiredAuthorization string, requestBody any) *ApiResponse[any] {
	result := &ApiResponse[any]{}

	targetUrl := client.GetServiceURL() + url

	jsonString, err := json.Marshal(requestBody)
	if err != nil {
		result = buildFailureInfo(err)
		return result
	}

	request, _ := http.NewRequest(verb, targetUrl, bytes.NewBuffer(jsonString))

	request.Header.Set("Content-Type", "application/json")
	fillAuthorizationHeader(client, requiredAuthorization, request)
	fillRequestIdHeaders(client, requestLog, request)

	httpClient := &http.Client{}
	response, err := httpClient.Do(request)
	if err != nil {
		result = buildFailureInfo(err)
	} else {
		data, _ := io.ReadAll(response.Body)
		result = buildResponse(data, client, response)
	}

	return result
}

func buildResponse(data []byte, client *BaseServiceClient, response *http.Response) *ApiResponse[any] {
	mapResult := make(map[string]interface{})
	err := json.Unmarshal(data, &mapResult)
	if err != nil {
		bodyType := "api_failure"
		return &ApiResponse[any]{
			Header: ResponseHeaderDTO{
				BodyType:        &bodyType,
				HttpStatusCode:  response.StatusCode,
				Code:            ERR_UNABLE_PARSE_SERVICE_RESPONSE,
				Message:         "The response got from the service is not valid JSON and can not be parsed: " + err.Error(),
				RequestId:       nil,
				ParentRequestId: nil,
			},
		}
	}

	headerMap := mapResult["header"].(map[string]any)
	responseType := headerMap["body_type"].(string)

	if generator, exists := client.TypeMapper[responseType]; exists {
		result := generator().(ApiResponse[interface{}])
		json.Unmarshal(data, &result)

		return &result
	} else if responseType == "empty" || responseType == "api_failure" {
		result := ApiResponse[interface{}]{}
		json.Unmarshal(data, &result)

		return &result
	} else if responseType == "request_error_info" {
		result := &ApiResponse[RequestValidationFailureDTO]{}
		json.Unmarshal(data, result)

		ret := ApiResponse[any]{
			Header: result.Header,
			Body:   result.Body,
		}

		return &ret
	} else {
		panic("Unexpected error occurred while parsing the API response.")
	}
}

func buildFailureInfo(err error) *ApiResponse[any] {
	bodyType := "api_failure"
	return &ApiResponse[any]{
		Header: ResponseHeaderDTO{
			BodyType:        &bodyType,
			HttpStatusCode:  -1,
			Code:            ERR_CONNECTION_FAILURE,
			Message:         err.Error(),
			RequestId:       nil,
			ParentRequestId: nil,
		},
	}
}

func fillRequestIdHeaders(client *BaseServiceClient, requestLog *RequestLog, request *http.Request) {
	if client.Mode == SERVICE_INTERNAL {
		request.Header.Set(REQUEST_ID_HEADER, uuid2.New().String())
		request.Header.Set(PARENT_REQUEST_ID_HEADER, requestLog.RequestIdentifier)
		request.Header.Set(CALLER_SERVICE_HEADER, client.ServiceName)
		request.Header.Set(CALLER_SERVICE_HASH_HEADER, client.ServiceVersionHash)
		request.Header.Set(CALLER_SERVICE_VERSION_HEADER, client.ServiceVersion)
	}
}

func fillAuthorizationHeader(client *BaseServiceClient, authorization string, request *http.Request) {
	if authorization == "@allow_all" || authorization == "@not_signed_in" {
		// No need to include an authorization header
		return
	} else if authorization == "@signed_in" {
		request.Header.Set("Authorization", "Bearer "+createJwt(client, nil, false))
	} else if authorization == "@service_call" {
		request.Header.Set("Authorization", "Bearer "+createJwt(client, nil, true))
	} else {
		request.Header.Set("Authorization", "Bearer "+createJwt(client, &authorization, false))
	}
}

func createJwt(client *BaseServiceClient, authority *string, isService bool) string {
	privateKey := DecodePrivate(client.SigningKey)
	claims := jwt.MapClaims{
		"iss": client.CallerServiceName,
		"sub": client.CallerServiceName,
		"exp": time.Now().Add(60 * time.Second).Unix(),
	}

	if isService {
		claims["is_inter_service_call"] = true
	}

	if authority != nil {
		claims["authorities"] = []string{*authority}
	}

	t := jwt.NewWithClaims(jwt.SigningMethodES512, claims)
	token, err := t.SignedString(privateKey)

	if err != nil {
		panic(err)
	}

	return token
}
