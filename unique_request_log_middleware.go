package lib

import (
	"github.com/gofiber/fiber/v2"
)

const LOCAL_KEY_FOR_REQUEST_LOG = "REQUEST_LOG_OBJECT"

type UniqueRequestLogMiddlewareConfig struct {
	IsInProduction bool
}

var uniqueRequestLogMiddlewareConfig *UniqueRequestLogMiddlewareConfig

func UniqueRequestLogMiddleware(context *fiber.Ctx) error {
	if uniqueRequestLogMiddlewareConfig == nil {
		mode := ServiceInstance.GetConfiguration("application.mode").(string)

		uniqueRequestLogMiddlewareConfig = &UniqueRequestLogMiddlewareConfig{
			IsInProduction: mode == "production",
		}
	}

	requestLog := &RequestLog{}
	defer requestLog.Save()
	context.Locals(LOCAL_KEY_FOR_REQUEST_LOG, requestLog)

	return context.Next()
}
