package lib

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"github.com/golang-jwt/jwt/v5"
	uuid2 "github.com/google/uuid"
	"io"
	"net/http"
	"strings"
	"time"
)

type IApiResponse interface {
	GetBodyType() string
}

type ApiResponse[Type any] struct {
	Header ResponseHeaderDTO `json:"header"`
	Body   Type              `json:"body"`
}

type ResponseHeaderDTO struct {
	BodyType        *string     `json:"body_type"`
	HttpStatusCode  int         `json:"status_code"`
	Code            string      `json:"code"`
	Message         string      `json:"message"`
	RequestId       *uuid2.UUID `json:"request_id"`
	ParentRequestId *string     `json:"parent_request_id"`
}

type AtomicValueDTO[Type any] struct {
	Value Type `json:"value"`
}

func IsSuccessful[Type any](response ApiResponse[Type]) bool {
	return response.Header.HttpStatusCode >= 200 && response.Header.HttpStatusCode < 300
}

func CreateInternalJWT(service *BaseConvergenceService, authorities []string, serviceJwt bool) string {
	signingKey := service.GetConfiguration("security.authentication.secret").(string)
	signingKey = strings.Replace(signingKey, "\\n", "\n", -1)
	privateKey := DecodePrivate(signingKey)
	now := time.Now()
	oneMinuteLater := now.Unix() + 60

	token := jwt.NewWithClaims(jwt.SigningMethodES512, jwt.MapClaims{
		"iss":                   service.ServiceName,
		"sub":                   service.ServiceName,
		"exp":                   oneMinuteLater,
		"authorities":           authorities,
		"is_inter_service_call": serviceJwt,
	})

	jwtString, err := token.SignedString(privateKey)
	if err != nil {
		panic(err)
	}

	return jwtString
}

func PostRequest[Type any](host string, endpoint string, payload any, jwt string, expectedCode int) ApiResponse[Type] {
	var result ApiResponse[Type]
	success := false

	for i := 0; i < 10; i++ {
		result = postRequestInternal[Type](host, endpoint, payload, jwt, expectedCode)
		if result.Header.HttpStatusCode != expectedCode {
			time.Sleep(time.Millisecond * 500)
		} else {
			success = true
			break
		}
	}

	if !success {
		failureInfo := "failure_info"
		result = ApiResponse[Type]{
			Header: ResponseHeaderDTO{
				HttpStatusCode: 500,
				Code:           "err_api_internal_error",
				Message:        "Unable to connect to the service after 5 retries.",
				BodyType:       &failureInfo,
			},
		}
	}

	return result
}

func postRequestInternal[Type any](host string, endpoint string, payload any, jwt string, expectedCode int) ApiResponse[Type] {
	url := host + endpoint

	jsonContent, err := json.Marshal(payload)
	if err != nil {
		panic(err)
	}

	client := &http.Client{}
	request, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonContent))
	if err != nil {
		panic(err)
	}

	if jwt != "" {
		request.Header.Set("Authorization", "Bearer "+jwt)
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := client.Do(request)
	if err != nil {
		panic(err)
	}

	responseBodyBytes, err := io.ReadAll(response.Body)

	if err != nil {
		panic(err)
	}

	result := ApiResponse[Type]{}
	err = json.Unmarshal(responseBodyBytes, &result)
	if err != nil {
		panic(err)
	}

	return result
}

func DecodePrivate(pemEncodedPriv string) *ecdsa.PrivateKey {
	blockPriv, _ := pem.Decode([]byte(pemEncodedPriv))

	x509EncodedPriv := blockPriv.Bytes

	privateKey, err := x509.ParseECPrivateKey(x509EncodedPriv)
	if err != nil {
		panic(err)
	}

	return privateKey
}

func (e AtomicValueDTO[Type]) GetBodyType() string {
	return "atomic_value"
}
