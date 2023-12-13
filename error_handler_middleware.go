package lib

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gofiber/fiber/v2"
	"runtime/debug"
)

type ErrorHandlerMiddlewareConfig struct {
	IsInProduction bool
}

type PanicRecoveryInfo struct {
	PanicHappened bool
	StackTrace    string
}

var errorHandlerMiddlewareConfig *ErrorHandlerMiddlewareConfig

func ErrorHandlerMiddleware(context *fiber.Ctx) error {
	if errorHandlerMiddlewareConfig == nil {
		mode := ServiceInstance.GetConfiguration("application.mode").(string)

		errorHandlerMiddlewareConfig = &ErrorHandlerMiddlewareConfig{
			IsInProduction: mode == "production",
		}
	}

	context.Locals("CONVERGENCE_REQUEST_PANIC_INFO", &PanicRecoveryInfo{PanicHappened: false})
	err := callNextHandler(context)

	panicInfo := context.Locals("CONVERGENCE_REQUEST_PANIC_INFO").(*PanicRecoveryInfo)
	if panicInfo.PanicHappened {
		requestLog := context.Locals(LOCAL_KEY_FOR_REQUEST_LOG).(*RequestLog)
		requestLog.Error("A panic occurred while executing the request.")
		requestLog.Exception(panicInfo.StackTrace)
		fmt.Println("----------------\nPanic stack trace:\n" + panicInfo.StackTrace + "\n----------------")
		err = errors.New("A panic occurred while executing the request.")
	}

	if err != nil {
		var managedError *ManagedApiError
		if errors.As(err, &managedError) {
			return SendManagedErrorResponse(context, managedError)
		} else if errorHandlerMiddlewareConfig.IsInProduction {
			fmt.Println("Unexpected error occurred, but won't send back to user.")
			fmt.Println(err)
			return SendUnmanagedErrorResponse(context, 500, "An unexpected error happened during API execution")
		} else {
			return SendUnmanagedErrorResponse(context, 500, err.Error())
		}
	}

	return nil
}

func callNextHandler(context *fiber.Ctx) error {
	defer func() {
		if r := recover(); r != nil {
			panicInfo := context.Locals("CONVERGENCE_REQUEST_PANIC_INFO").(*PanicRecoveryInfo)
			panicInfo.PanicHappened = true
			panicInfo.StackTrace = string(debug.Stack())
		}
	}()

	err := context.Next()
	return err
}

func SendManagedErrorResponse(context *fiber.Ctx, err *ManagedApiError) error {
	statusCode := err.HttpStatusCode
	context.Status(statusCode)
	bodyType := err.bodyType

	response := ApiResponse[any]{
		Header: ResponseHeaderDTO{
			BodyType:        bodyType,
			HttpStatusCode:  statusCode,
			Code:            err.Code,
			Message:         err.Message,
			RequestId:       err.RequestId,
			ParentRequestId: err.ParentRequestId,
		},
		Body: nil,
	}

	if err.HasCustomBody() {
		response.Body, response.Header.BodyType = err.CustomBody()
	}

	jsonString, _ := json.Marshal(response)

	context.Set("Content-Type", "application/json")
	return context.SendString(string(jsonString))
}

func SendUnmanagedErrorResponse(context *fiber.Ctx, statusCode int, message string) error {
	requestLog := InitializeRequestLogForGatewayMiddleware(context)

	context.Status(statusCode)
	bodyType := "api_failure"

	response := ApiResponse[any]{
		Header: ResponseHeaderDTO{
			BodyType:        &bodyType,
			HttpStatusCode:  statusCode,
			Code:            API_INTERNAL_ERROR,
			Message:         message,
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
