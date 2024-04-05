package lib

import (
	"encoding/json"
	"fmt"
	"github.com/gofiber/fiber/v2"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gorm_logger "gorm.io/gorm/logger"
	"reflect"
	"time"
)

type ConvergenceHandler = func() (any, string, error)

func RunApiMethod[Type any](requestLog *RequestLog, context *fiber.Ctx, function ConvergenceHandler) error {
	body, bodyType, err := function()

	if err != nil {
		return err
	}

	if body != nil {
		rt := reflect.TypeOf(body)
		if rt.Kind() == reflect.Slice || rt.Kind() == reflect.Array {
			bodyType = "list[" + bodyType + "]"
		}
	} else {
		bodyType = "empty"
	}

	response := ApiResponse[Type]{
		Header: ResponseHeaderDTO{
			BodyType:        &bodyType,
			HttpStatusCode:  200,
			Code:            "",
			Message:         "",
			RequestId:       requestLog.GetRawRequestID(),
			ParentRequestId: requestLog.ParentRequestIdentifier,
		},
	}

	if body != nil {
		response.Body = body.(Type)
	}

	jsonString, err := json.Marshal(response)
	if err != nil {
		panic(err)
	}

	context.Status(response.Header.HttpStatusCode)
	context.Set("Content-Type", "application/json")

	FinishRequestLog(requestLog, &response)

	return context.SendString(string(jsonString))
}

func MakeApiManagedErrorResponse(managedError *ManagedApiError, context *fiber.Ctx) error {
	bodyType := "api_failure"

	response := ApiResponse[any]{
		Header: ResponseHeaderDTO{
			BodyType:        &bodyType,
			HttpStatusCode:  managedError.HttpStatusCode,
			Code:            managedError.Code,
			Message:         managedError.Message,
			RequestId:       managedError.RequestId,
			ParentRequestId: managedError.ParentRequestId,
		},
		Body: nil,
	}

	if managedError.HasCustomBody() {
		response.Body, response.Header.BodyType = managedError.CustomBody()
	}

	jsonString, err := json.Marshal(response)
	if err != nil {
		panic(err)
	}

	context.Status(response.Header.HttpStatusCode)
	context.Set("Content-Type", "application/json")
	return context.SendString(string(jsonString))
}

func MakeErrorResponse(requestLog *RequestLog, message string, context *fiber.Ctx) error {
	bodyType := "api_failure"

	response := ApiResponse[any]{
		Header: ResponseHeaderDTO{
			BodyType:        &bodyType,
			HttpStatusCode:  500,
			Code:            API_INTERNAL_ERROR,
			Message:         message,
			RequestId:       requestLog.GetRawRequestID(),
			ParentRequestId: requestLog.ParentRequestIdentifier,
		},
		Body: nil,
	}

	jsonString, err := json.Marshal(response)
	if err != nil {
		panic(err)
	}

	context.Status(response.Header.HttpStatusCode)
	context.Set("Content-Type", "application/json")
	return context.SendString(string(jsonString))
}

func GetGormConnection() (*gorm.DB, error) {
	var err error
	var db *gorm.DB

	host := ServiceInstance.GetConfiguration("database.host")
	port := ServiceInstance.GetConfiguration("database.port")
	user := ServiceInstance.GetConfiguration("database.username")
	password := ServiceInstance.GetConfiguration("database.password")
	databaseName := ServiceInstance.GetConfiguration("database.name")
	connectionString := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, databaseName)

	if db, err = gorm.Open(postgres.Open(connectionString), makeGormConfiguration()); err != nil {
		return nil, err
	}

	return db, nil
}

func makeGormConfiguration() *gorm.Config {
	return &gorm.Config{
		Logger: gorm_logger.Default.LogMode(gorm_logger.Silent),
		NowFunc: func() time.Time {
			return *UtcNow()
		},
	}
}

func CloseGormConnection(db *gorm.DB) {
	dbInstance, _ := db.DB()
	_ = dbInstance.Close()
}
