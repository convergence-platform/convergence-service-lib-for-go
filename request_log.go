package lib

import (
	"encoding/json"
	"fmt"
	"github.com/gofiber/fiber/v2"
	uuid2 "github.com/google/uuid"
	"os"
	"reflect"
	"strings"
)

const REQUEST_ID_HEADER = "X-CONVERGENCE-REQUEST-ID"
const PARENT_REQUEST_ID_HEADER = "X-CONVERGENCE-PARENT-REQUEST-ID"
const CALLER_SERVICE_HEADER = "X-CONVERGENCE-CALLER-SERVICE"
const CALLER_SERVICE_HASH_HEADER = "X-CONVERGENCE-CALLER-SERVICE-HASH"
const CALLER_SERVICE_VERSION_HEADER = "X-CONVERGENCE-CALLER-SERVICE-VERSION"

type LogEntryServiceInfo struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	VersionHash string `json:"hash"`
}

type LogEntry struct {
	Timestamp      int64  `json:"timestamp"`
	Level          string `json:"level"`
	Message        string `json:"message"`
	Type           string `json:"type"`
	Arguments      any    `json:"arguments"`
	NamedArguments any    `json:"named_arguments"`
	ThreadID       int    `json:"thread_id"`
}

type RequestLog struct {
	RequestIdentifier       string               `json:"request_identifier"`
	ParentRequestIdentifier *string              `json:"parent_request_identifier"`
	CallerService           *LogEntryServiceInfo `json:"caller_service"`
	ReceiverService         *LogEntryServiceInfo `json:"receiver_service"`
	StartTimestamp          int64                `json:"start_timestamp"`
	EndTimestamp            int64                `json:"end_timestamp"`
	Headers                 map[string]string    `json:"headers"`
	URL                     string               `json:"url"`
	Parameters              []any                `json:"parameters"`
	LogEntries              []LogEntry           `json:"log_entries"`
	Response                any                  `json:"response"`
	rawRequestID            *uuid2.UUID          `json:"-"`
	logTypePrefix           string               `json:"-"`
}

func InitializeRequestLogForProcessingQueue(logPrefix string, requestIdentifier *uuid2.UUID, parentRequestIdentifier *string, queueName string, version string, versionHash string) *RequestLog {
	queueInfo := LogEntryServiceInfo{
		Name:        queueName,
		Version:     ServiceInstance.ServiceVersion,
		VersionHash: ServiceInstance.ServiceVersionHash,
	}

	return &RequestLog{
		RequestIdentifier:       strings.ToLower(logPrefix) + "_" + requestIdentifier.String(),
		ParentRequestIdentifier: parentRequestIdentifier,
		CallerService:           loadCurrentService(),
		ReceiverService:         &queueInfo,
		StartTimestamp:          UtcNow().UnixMilli(),
		EndTimestamp:            0,
		Headers:                 nil,
		URL:                     "",
		Parameters:              nil,
		LogEntries:              []LogEntry{},
		Response:                nil,
		rawRequestID:            requestIdentifier,
		logTypePrefix:           logPrefix,
	}
}

func InitializeRequestLogForGatewayMiddleware(context *fiber.Ctx) *RequestLog {
	r, _ := initializeRequestLogFromContext(context, false)
	return r
}

func InitializeRequestLog(context *fiber.Ctx, parameters ...any) (*RequestLog, error) {
	r, e := initializeRequestLogFromContext(context, true, parameters...)
	return r, e
}

func initializeRequestLogFromContext(context *fiber.Ctx, errorOnFailure bool, parameters ...any) (*RequestLog, error) {
	service := ServiceInstance
	var isBehindGateway bool = service.GetBooleanConfiguration("security.is_behind_gateway")
	if isBehindGateway && errorOnFailure {
		err := validateBehindGatewayHeaders(context)
		if err != nil {
			return nil, err
		}
	}

	result := context.Locals(LOCAL_KEY_FOR_REQUEST_LOG).(*RequestLog)
	var err error

	result.StartTimestamp = UtcNow().UnixMilli()
	result.Headers = loadRequestHeaders(context)
	result.RequestIdentifier, result.rawRequestID, err = getRequestIDFromHeader(result, isBehindGateway, context)
	if err != nil && errorOnFailure {
		return nil, err
	}
	result.ParentRequestIdentifier = loadParentIdentifier(result)
	result.CallerService = loadCallerService(result.Headers)
	result.ReceiverService = loadCurrentService()
	result.URL = context.OriginalURL()
	result.Parameters = parameters
	result.logTypePrefix = ServiceInstance.GetConfiguration("observability.request_id_prefix").(string)

	delete(result.Headers, strings.ToLower(REQUEST_ID_HEADER))
	delete(result.Headers, strings.ToLower(PARENT_REQUEST_ID_HEADER))
	delete(result.Headers, strings.ToLower(CALLER_SERVICE_HEADER))
	delete(result.Headers, strings.ToLower(CALLER_SERVICE_HASH_HEADER))
	delete(result.Headers, strings.ToLower(CALLER_SERVICE_VERSION_HEADER))
	removeJwtSignatureForLogging(&result.Headers)

	return result, nil
}

func removeJwtSignatureForLogging(m *map[string]string) {
	if authHeader, ok := (*m)["authorization"]; ok {
		authHeader = authHeader[0 : strings.LastIndex(authHeader, ".")+1]
		authHeader += "***********"
		(*m)["authorization"] = authHeader
	}
}

func loadRequestHeaders(context *fiber.Ctx) map[string]string {
	headers := context.GetReqHeaders()
	result := make(map[string]string)

	for k, v := range headers {
		result[strings.ToLower(k)] = v[0]
	}

	return result
}

func getRequestIDFromHeader(result *RequestLog, isBehindGateway bool, context *fiber.Ctx) (string, *uuid2.UUID, error) {
	header := strings.ToLower(REQUEST_ID_HEADER)
	var requestUuid *uuid2.UUID

	if !isBehindGateway {
		headerValue := getGatewayHeaderFromRequest(header, context)
		if headerValue != nil {
			error := ConstructManagedApiError()
			error.HttpStatusCode = BAD_REQUEST
			error.Code = INVALID_DATA
			error.Message = "The request has unexpected request ID header."
			requestId, _ := uuid2.Parse(*headerValue)
			error.RequestId = &requestId

			return "", nil, error
		}

		if result.rawRequestID != nil {
			// This was already initialized, we don't want to change the generated UUID
			requestUuid = result.rawRequestID
		} else {
			requestUuidVal, _ := uuid2.NewRandom()
			requestUuid = &requestUuidVal
		}
	} else {
		requestUuidVal, _ := uuid2.Parse(result.Headers[header])
		requestUuid = &requestUuidVal
	}

	requestType := ServiceInstance.GetConfiguration("observability.request_id_prefix")
	return strings.ToLower(requestType.(string)) + "_" + requestUuid.String(), requestUuid, nil
}

func validateBehindGatewayHeaders(context *fiber.Ctx) error {
	missingHeaders := make([]string, 0)

	if header := getGatewayHeaderFromRequest(REQUEST_ID_HEADER, context); header == nil {
		missingHeaders = append(missingHeaders, REQUEST_ID_HEADER)
	}

	if header := getGatewayHeaderFromRequest(CALLER_SERVICE_HEADER, context); header == nil {
		missingHeaders = append(missingHeaders, CALLER_SERVICE_HEADER)
	}

	if header := getGatewayHeaderFromRequest(CALLER_SERVICE_HASH_HEADER, context); header == nil {
		missingHeaders = append(missingHeaders, CALLER_SERVICE_HASH_HEADER)
	}

	if header := getGatewayHeaderFromRequest(CALLER_SERVICE_VERSION_HEADER, context); header == nil {
		missingHeaders = append(missingHeaders, CALLER_SERVICE_VERSION_HEADER)
	}

	if len(missingHeaders) > 0 {
		details := RequestValidationFailureDTO{}
		for _, header := range missingHeaders {
			missingHeader := &RequestValidationFieldFailureDTO{}
			missingHeader.Field = header
			missingHeader.Location = "header"
			missingHeader.Messages = append(missingHeader.Messages, "Request is missing header.")

			details.Errors = append(details.Errors, missingHeader)
		}

		err := ConstructManagedApiError()
		err.HttpStatusCode = GATEWAY_ERROR
		err.Code = INVALID_DATA
		err.Message = "The request input was invalid, missing the mandatory API gateway headers."
		err.SetBody(details, "request_error_info")

		return err
	}

	return nil
}

func getGatewayHeaderFromRequest(header string, context *fiber.Ctx) *string {
	headers := context.GetReqHeaders()
	var value *string = nil

	for k, v := range headers {
		if strings.ToUpper(k) == strings.ToUpper(header) {
			value = &v[0]
			break
		}
	}

	return value
}

func loadParentIdentifier(result *RequestLog) *string {
	if value, ok := result.Headers[strings.ToLower(PARENT_REQUEST_ID_HEADER)]; ok {
		return &value
	}

	return nil
}

func loadCallerService(headers map[string]string) *LogEntryServiceInfo {
	result := LogEntryServiceInfo{
		Name:        headers[strings.ToLower(CALLER_SERVICE_HEADER)],
		Version:     headers[strings.ToLower(CALLER_SERVICE_VERSION_HEADER)],
		VersionHash: headers[strings.ToLower(CALLER_SERVICE_HASH_HEADER)],
	}

	return &result
}

func loadCurrentService() *LogEntryServiceInfo {
	result := LogEntryServiceInfo{
		Name:        ServiceInstance.ServiceName,
		Version:     ServiceInstance.ServiceVersion,
		VersionHash: ServiceInstance.ServiceVersionHash,
	}

	return &result
}

func FinishRequestLog[Type any](r *RequestLog, response *ApiResponse[Type]) {
	r.Response = response
}

func (r *RequestLog) Info(message string, arguments ...any) {
	addLogEntry(r, "info", message, arguments...)
}

func (r *RequestLog) Error(message string, arguments ...any) {
	addLogEntry(r, "error", message, arguments...)
}

func (r *RequestLog) Warning(message string, arguments ...any) {
	addLogEntry(r, "warning", message, arguments...)
}

func (r *RequestLog) Debug(message string, arguments ...any) {
	addLogEntry(r, "debug", message, arguments...)
}

func (r *RequestLog) Exception(stackTrace string) {
	details := map[string]string{
		"format":      "go: stack_trace",
		"stack_trace": stackTrace,
	}

	entry := LogEntry{
		Timestamp:      UtcNow().UnixMilli(),
		Level:          "exception",
		Message:        "Go panic occurred.",
		Arguments:      nil,
		NamedArguments: details,
		Type:           "exception_entry",
		ThreadID:       -1,
	}

	r.LogEntries = append(r.LogEntries, entry)
}

func (r *RequestLog) Save() {
	r.EndTimestamp = UtcNow().UnixMilli()

	if r.rawRequestID != nil {
		mapped := convertObjectToDictionary(r).(map[string]any)
		mapped["service_language"] = "go"
		mapped["template_version"] = LIBRARY_VERSION

		jsonString, err := json.Marshal(mapped)
		if err != nil {
			panic(err.Error())
		} else {
			folder := ServiceInstance.GetConfiguration("observability.path")

			filePath := fmt.Sprintf("%s/%s_%s.crl", folder, strings.ToLower(r.logTypePrefix), r.GetRawRequestID().String())
			if err := os.WriteFile(filePath, []byte(jsonString), 0600); err != nil {
				panic(err.Error())
			}
		}
	}
}

func convertObjectToDictionary(in any) any {
	if in == nil {
		return nil
	}

	v := reflect.ValueOf(in)
	kind := v.Kind()
	if kind == reflect.Ptr {
		v = v.Elem()
		kind = v.Kind()
	}

	if kind == reflect.Slice {
		result := make([]any, 0)
		s := reflect.ValueOf(in)

		for i := 0; i < s.Len(); i++ {
			result = append(result, convertObjectToDictionary(s.Index(i).Interface()))
		}

		return result
	} else if kind == reflect.Map {
		result := make(map[string]any)

		for _, k := range v.MapKeys() {
			value := v.MapIndex(k).Interface()
			result[k.String()] = convertObjectToDictionary(value)
		}

		return result
	} else if kind != reflect.Struct {
		return in
	}

	out := make(map[string]any)

	typ := v.Type()
	for i := 0; i < v.NumField(); i++ {
		// gets us a StructField
		fi := typ.Field(i)

		if tagv := fi.Tag.Get("json"); tagv != "" {
			if tagv == "-" {
				continue
			}
			if sensitiveValue := fi.Tag.Get("convergence_sensitive"); sensitiveValue == "true" {
				out[tagv] = "***********"
			} else {
				out[tagv] = convertObjectToDictionary(v.Field(i).Interface())
			}
		}
	}
	return out
}

func (r *RequestLog) GetRawRequestID() *uuid2.UUID {
	return r.rawRequestID
}

func (r *RequestLog) SetRawRequestID(id *uuid2.UUID) {
	r.rawRequestID = id
}

func addLogEntry(r *RequestLog, level string, message string, arguments ...any) {
	entry := LogEntry{
		Timestamp:      UtcNow().UnixMilli(),
		Level:          level,
		Message:        message,
		Arguments:      arguments,
		NamedArguments: nil,
		Type:           "log_entry",
		ThreadID:       -1,
	}

	r.LogEntries = append(r.LogEntries, entry)
}
