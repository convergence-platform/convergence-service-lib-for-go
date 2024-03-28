package lib

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"github.com/convergence-platform/convergence-service-lib-for-go/db_migrations"
	"github.com/convergence-platform/convergence-service-lib-for-go/db_migrations/postgres"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/golang-jwt/jwt/v5"
	uuid2 "github.com/google/uuid"
	_ "github.com/lib/pq"
	"gopkg.in/yaml.v3"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"
)

type EndpointAuthorizationHandler = func(*fiber.Ctx, *jwt.Token, bool) *string

var ServiceInstance *BaseConvergenceService
var OverrideServiceProfile *string

type ServiceAuthorityDeclaration struct {
	UUID        uuid2.UUID
	Authority   string
	DisplayName string
	Tier        int
}

type BaseConvergenceService struct {
	endpointsInfo          []ConvergenceEndpointInfo
	configuration          map[string]any
	ServiceState           ServiceState
	Migrations             []any
	Authorities            []ServiceAuthorityDeclaration
	ServiceName            string
	ServiceVersion         string
	ServiceVersionHash     string
	Fiber                  *fiber.App
	Endpoints              []*ServiceEndpointInfoDTO
	endpointsAuthorization []*ServiceEndpointAuthorizationDetails
}

type ServiceEndpointAuthorizationDetails struct {
	URL           string
	Method        string
	Authorization func(context *fiber.Ctx, token *jwt.Token, hadAuthorizationHeader bool) *string
}

func ConstructConvergenceService(service *BaseConvergenceService, configurations embed.FS) {
	if ServiceInstance != nil {
		panic("There can only be one instance/extension of ConvergenceService.")
	}

	ServiceInstance = service
	service.ServiceState = ServiceState{Status: "initializing"}
	service.Endpoints = []*ServiceEndpointInfoDTO{}

	service.configuration = loadServiceConfiguration(configurations)
}

func getServiceProfile() string {
	if OverrideServiceProfile != nil {
		return *OverrideServiceProfile
	}

	args := os.Args[1:]

	for i, arg := range args {
		if arg == "--profile" && i+1 < len(args) {
			return args[i+1]
		}
	}

	// no --profile, check env variable
	if profile := os.Getenv("CONVERGENCE_SERVICE_PROFILE"); profile != "" {
		return profile
	}

	return "default"
}

func (service *BaseConvergenceService) ConfigurationExists(path string) bool {
	parts := strings.Split(path, ".")
	var config = service.configuration

	for i, part := range parts {
		if i == len(parts)-1 {
			if _, exists := config[part]; !exists {
				return false
			}
		} else {
			if _, exists := config[part]; !exists {
				return false
			}
			temp := config[part]
			config = temp.(map[string]any)
		}
	}

	return true
}

func (service *BaseConvergenceService) GetConfiguration(path string) any {
	parts := strings.Split(path, ".")
	var result any
	var config = service.configuration

	for i, part := range parts {
		if i == len(parts)-1 {
			result = config[part]
		} else {
			temp := config[part]
			config = temp.(map[string]any)
		}
	}

	return result
}

func (service *BaseConvergenceService) GetIntegerConfiguration(path string) int {
	value := service.GetConfiguration(path)

	if _, ok := value.(int); ok {
		return value.(int)
	}

	panic("The config path " + path + " is not an integer")
}

func (service *BaseConvergenceService) GetBooleanConfiguration(path string) bool {
	value := service.GetConfiguration(path)

	if _, ok := value.(bool); ok {
		return value.(bool)
	}

	panic("The config path " + path + " is not a boolean")
}

func (service *BaseConvergenceService) Initialize() {
	service.Fiber = fiber.New(fiber.Config{
		Prefork:               false,
		CaseSensitive:         true,
		StrictRouting:         true,
		DisableStartupMessage: true,
		AppName:               service.ServiceName + " " + service.ServiceVersion,
	})

	printFiglet()
	printServerPort(service)

	if isDatabaseEnabled(service) {
		service.ServiceState.Status = "initializing_db"
		migrateDatabase(service)
		service.ServiceState.Status = "db_initialized"
	} else {
		fmt.Println("Service is configured to disable database initialization.")
	}
	saveServiceAuthorities(service)
	service.ServiceState.Status = "initializing_service"
	initializeCors()
	initializeServiceMiddleware(service)
	service.ServiceState.Status = "healthy"

}

func isDatabaseEnabled(service *BaseConvergenceService) bool {
	result := true

	if service.ConfigurationExists("database.disable") {
		result = !service.GetBooleanConfiguration("database.disable")
	}

	return result
}

func (service *BaseConvergenceService) Start() {
	fmt.Println("Launching service with info:")
	fmt.Println("   Name: " + service.ServiceName)
	fmt.Println("   Version: " + service.ServiceVersion)
	fmt.Println("   Hash: " + service.ServiceVersionHash)
	fmt.Println("")

	port := fmt.Sprintf("%d", service.GetConfiguration("server.port"))
	service.Fiber.Listen(":" + port)
}

func (service *BaseConvergenceService) GetStatus() ServiceState {
	return service.ServiceState
}

func loadConfigurationFile(configurations embed.FS, profile string) map[string]any {
	fileName := "configurations/application"
	if profile != "default" {
		fileName += "-" + profile
	}

	fileName += ".yaml"
	yamlString, err := configurations.ReadFile(fileName)
	if err != nil {
		panic(err)
	}

	obj := make(map[string]any)
	err = yaml.Unmarshal(yamlString, obj)

	if err != nil {
		panic(err)
	}

	return obj
}

func loadServiceConfiguration(configurations embed.FS) map[string]any {
	profile := getServiceProfile()

	defaultConfiguration := loadConfigurationFile(configurations, "default")
	profileConfiguration := make(map[string]any)
	if profile != "default" {
		profileConfiguration = loadConfigurationFile(configurations, profile)
	}

	mergedConfigurations := mergeConfigurations(defaultConfiguration, profileConfiguration)
	return swapEnvironmentVariables(mergedConfigurations)
}

func swapEnvironmentVariables(configurations map[string]any) map[string]any {
	result := make(map[string]any)
	for k, v := range configurations {
		if _, ok := v.(map[string]any); ok {
			result[k] = swapEnvironmentVariables(v.(map[string]any))
		} else if _, ok := v.(string); ok {
			stringValue := v.(string)
			if strings.HasPrefix(stringValue, "${") && strings.HasSuffix(stringValue, "}") {
				stringValue = stringValue[2 : len(stringValue)-1]
				result[k] = os.Getenv(stringValue)
			} else {
				result[k] = v
			}
		} else {
			result[k] = v
		}
	}

	return result
}

func mergeConfigurations(resultConfig map[string]any, profileConfig map[string]any) map[string]any {
	for k, v := range profileConfig {
		_, exists := resultConfig[k]
		if exists {
			if _, ok := v.(map[string]any); ok {
				resultConfig[k] = mergeConfigurations(resultConfig[k].(map[string]any), v.(map[string]any))
			} else {
				resultConfig[k] = v
			}
		} else {
			resultConfig[k] = v
		}
	}

	return resultConfig
}

func printFiglet() {
	fmt.Println("   ______")
	fmt.Println("  / ____/___  ____ _   _____  _________ ____  ____  ________")
	fmt.Println(" / /   / __ \\/ __ \\ | / / _ \\/ ___/ __ `/ _ \\/ __ \\/ ___/ _ \\")
	fmt.Println("/ /___/ /_/ / / / / |/ /  __/ /  / /_/ /  __/ / / / /__/  __/")
	fmt.Println("\\____/\\____/_/ /_/|___/\\___/_/   \\__, /\\___/_/ /_/\\___/\\___/")
	fmt.Println("                                /____/")
	fmt.Println("")
	fmt.Println(" :: Convergence Platform ::")
	fmt.Println("     Version: " + LIBRARY_VERSION)
	fmt.Println("     Hash: " + LIBRARY_VERSION_HASH)
	fmt.Println("     Build Date: " + LIBRARY_BUILD_DATE)
	fmt.Println("")

}

func printServerPort(service *BaseConvergenceService) {
	fmt.Println("Server will run on port:", service.GetConfiguration("server.port"))
}

func migrateDatabase(service *BaseConvergenceService) {
	migrations := service.Migrations

	connectionString := GetDbConnectionString(service)
	connection, err := OpenDatabaseConnection(connectionString)
	if err != nil {
		panic(err)
	}
	defer connection.Close()

	fmt.Println("Starting to apply database migrations:")
	appliedDatabaseMigrations := getListOfMigrations(connection)

	error := ""
	failedMigration := ""
	for _, migration := range migrations {
		migrationName := getMigrationName(migration)
		if !slices.Contains(appliedDatabaseMigrations, migrationName) {
			if error != "" {
				fmt.Println("   - " + pad(migrationName+":", 60) + "[ SKIPPED ]")
			} else {
				migrationError := applyDatabaseMigration(migration, connection)
				if migrationError != "" {
					failedMigration = migrationName
					error = migrationError
					fmt.Println("   - " + pad(migrationName+":", 60) + "[ FAILED ]")
				} else {
					fmt.Println("   - " + pad(migrationName+":", 60) + "[ SUCCESS ]")
				}
			}
		} else {
			fmt.Println("   - " + pad(migrationName+":", 60) + "[ ALREADY_APPLIED ]")
		}
	}
	if error != "" {
		panic(failedMigration + " failed with error: " + error)
	}
}

func GetDbConnectionString(service *BaseConvergenceService) string {
	host := service.GetConfiguration("database.host")
	port := service.GetConfiguration("database.port")
	user := service.GetConfiguration("database.username")
	password := service.GetConfiguration("database.password")
	databaseName := service.GetConfiguration("database.name")
	connectionString := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, databaseName)
	return connectionString
}

func OpenDatabaseConnection(connectionString string) (*sql.DB, error) {
	i := 0
	for i < 30 {
		connection, err := sql.Open("postgres", connectionString)
		if err == nil && checkConnectionIsValid(connection) {
			return connection, nil
		} else {
			fmt.Println("Unable to connect to database, waiting a bit before retrying.")
			time.Sleep(500 * time.Millisecond)
		}
		i++
	}

	return nil, errors.New("Unable to connect to the database after 30 seconds")
}

func applyDatabaseMigration(migration any, connection *sql.DB) string {
	error := ""
	if casted, ok := migration.(db_migrations.DatabaseSeeds); ok {
		command := postgres.PostgresSeedToSQL(casted.Seeds)
		_, err := connection.Exec(command)
		if err != nil {
			error = err.Error()
		}

		if err == nil {
			error = saveMigrationState(connection, migration, command)
		}
	} else if casted, ok := migration.(db_migrations.DatabaseMigration); ok {
		command := postgres.PostgresTableToSQL(casted.MigrationDDL)
		_, err := connection.Exec(command)
		if err != nil && !casted.AllowFailure {
			error = err.Error()
		}

		if err == nil {
			error = saveMigrationState(connection, migration, command)
		}
	} else {
		error = "The migration type for " + getMigrationName(migration) + " is not supported"
	}

	return error
}

func saveMigrationState(connection *sql.DB, migration any, command string) string {
	migrationName := getMigrationName(migration)
	uuid := uuid2.New()
	query := "INSERT INTO database_migrations(uuid, migration_name, command, applied_timestamp) VALUES($1, $2, $3, CURRENT_TIMESTAMP)"

	_, err := connection.Exec(query, uuid, migrationName, command)

	if err != nil {
		return err.Error()
	}
	return ""
}

func pad(s string, length int) string {
	for len(s) < length {
		s = s + " "
	}

	return s
}

func getMigrationName(migration any) string {
	result := ""

	if casted, ok := migration.(db_migrations.DatabaseMigration); ok {
		result = casted.Name
	} else if casted, ok := migration.(db_migrations.DatabaseSeeds); ok {
		result = casted.Name
	}

	return result
}

func getListOfMigrations(connection *sql.DB) []string {
	rows, err := connection.Query("SELECT migration_name FROM database_migrations")
	if err != nil {
		panic(err)
	}
	defer rows.Close()
	result := []string{}
	for rows.Next() {
		var migrationName string
		err := rows.Scan(&migrationName)
		if err != nil {
			panic(err)
		}
		result = append(result, migrationName)
	}
	err = rows.Err()
	if err != nil {
		panic(err)
	}

	return result
}

func checkConnectionIsValid(connection *sql.DB) bool {
	rows, err := connection.Query("SELECT 1")
	if err != nil {
		return false
	}

	defer rows.Close()
	return true
}

func saveServiceAuthorities(service *BaseConvergenceService) {
	service.ServiceState.Status = "initializing_authorities"
	mode := service.GetConfiguration("application.mode")

	if mode == "production" {
		if len(service.Authorities) > 0 {
			fmt.Println("Service is running in production mode, starting to initialize authorities:")
			infrastructureService := InfrastructureMicroService{Service: service}
			authenticationServiceURL := infrastructureService.GetServiceURL("authentication-service")
			fmt.Println("   -> Authentication Service URL: " + authenticationServiceURL)
			authenticationService := AuthenticationMicroService{
				Service: service,
				URL:     authenticationServiceURL,
			}

			anyFailed := false
			for _, authority := range service.Authorities {
				if !registerSingleServiceAuthority(authority, authenticationService) {
					fmt.Println("   * " + pad(authority.Authority+":", 50) + " [ FAILED INITIALIZATION ]")
					anyFailed = true

				} else {
					fmt.Println("   * " + pad(authority.Authority+":", 50) + " [ INITIALIZED ]")
				}
			}

			if anyFailed {
				panic("Service was unable to initialize the necessary authorities.")
			}
		} else {
			fmt.Println("Service is running in production mode, but doesn't declare any authority")
		}
	} else {
		fmt.Println("Service is running in non-production mode, skipping the authorities initialization:")
		if len(service.Authorities) == 0 {
			fmt.Println("   -> Service doesn't declares any authority")
		} else {
			for _, authority := range service.Authorities {
				tier := pad("[Tier "+strconv.Itoa(authority.Tier)+"]", 10)
				fmt.Println("   - " + tier + " " + authority.Authority)
			}
		}
	}

	service.ServiceState.Status = "authorities_initialized"
}

func registerSingleServiceAuthority(authority ServiceAuthorityDeclaration, authenticationService AuthenticationMicroService) bool {
	request := RegisterServiceAuthorityRequestDTO{
		UUID:        authority.UUID,
		Authority:   authority.Authority,
		DisplayName: authority.DisplayName,
		Tier:        authority.Tier,
	}

	response := authenticationService.RegisterServiceAuthority(request)
	return IsSuccessful(response) && response.Body.Value
}

func initializeCors() {

}

func initializeServiceMiddleware(service *BaseConvergenceService) {
	service.Fiber.Use(logger.New())
	service.Fiber.Use(UniqueRequestLogMiddleware)
	service.Fiber.Use(ErrorHandlerMiddleware)
	service.Fiber.Use(GatewayHeaderValidationMiddleware)
	service.Fiber.Use(AuthorizationMiddleware)
}

func (service *BaseConvergenceService) RegisterRoute(method string,
	route string,
	handler fiber.Handler,
	expectedAuthorizationType string,
	exposedThroughGateway bool,
	maxPayloadSize string,
	timeout string,
	maintenanceMode string,
	rateLimitingPolicies []string,
	accepts []string) {
	method = strings.ToUpper(method)

	endpoint := &ServiceEndpointInfoDTO{
		URL:                       route,
		Method:                    method,
		ExposedThroughGateway:     exposedThroughGateway,
		AuthorizationTypeExpected: expectedAuthorizationType,
		MaxPayloadSize:            parseMaxPayloadSize(maxPayloadSize),
		Timeout:                   parseTimeout(timeout),
		RateLimitingPolicy:        parseRateLimitingPolicy(rateLimitingPolicies),
		Accepts:                   accepts,
		MaintenanceMode:           maintenanceMode,
	}

	service.Endpoints = append(service.Endpoints, endpoint)

	if method == "GET" {
		service.Fiber.Get(formatParamsFromBraceToColon(route), handler)
	} else if method == "POST" {
		service.Fiber.Post(formatParamsFromBraceToColon(route), handler)
	} else if method == "DELETE" {
		service.Fiber.Delete(formatParamsFromBraceToColon(route), handler)
	} else if method == "PATCH" {
		service.Fiber.Patch(formatParamsFromBraceToColon(route), handler)
	} else if method == "PUT" {
		service.Fiber.Put(formatParamsFromBraceToColon(route), handler)
	} else if method == "HEAD" {
		panic("The method HEAD is not supported as it may conflict with CORS.")
	} else {
		panic("The method " + method + " is not recognized")
	}

	service.endpointsAuthorization = append(service.endpointsAuthorization, &ServiceEndpointAuthorizationDetails{
		URL:           route,
		Method:        method,
		Authorization: getAuthorizationHandlerFor(expectedAuthorizationType),
	})
}

func parseRateLimitingPolicy(policies []string) []ConvergenceEndpointRateLimitPolicy {
	result := []ConvergenceEndpointRateLimitPolicy{}

	for _, p := range policies {
		parts := strings.Split(p, ":")
		if len(parts) != 3 {
			panic("The rate limit policy " + p + " is not valid.")
		}

		if parts[0] != "max_globally" && parts[0] != "max_per_session" && parts[0] != "max_per_ip" {
			panic("The rate limit policy " + p + " is not valid.")
		}
		c := ConvergenceEndpointRateLimitPolicy{}
		c.Policy = parts[0]
		if count, err := strconv.Atoi(parts[1]); err == nil {
			c.Count = count
		} else {
			panic("The rate limit policy " + p + " is not valid.")
		}

		duration := parts[2]
		coeff := 0
		units := make(map[string]int)
		units["s"] = 1
		units["m"] = 60
		units["h"] = 3600

		validUnit := false
		for k, v := range units {
			if strings.HasSuffix(duration, k) {
				validUnit = true
				coeff = v
			}
		}

		if !validUnit {
			panic("The rate limit policy " + p + " is not valid.")
		}

		value := duration[0 : len(duration)-1]
		if interval, err := strconv.Atoi(value); err != nil {
			panic("The rate limit policy " + p + " is not valid.")
		} else {
			c.Duration = interval * coeff
		}

		result = append(result, c)
	}

	return result
}

func parseTimeout(timeout string) int {
	if len(timeout) <= 2 {
		panic("The timeout " + timeout + " is not valid.")
	}

	if strings.HasSuffix(timeout, "ms") {
		value := timeout[0 : len(timeout)-2]
		if v, err := strconv.Atoi(value); err != nil {
			panic("The timeout " + timeout + " is not valid.")
		} else {
			return v
		}
	} else if strings.HasSuffix(timeout, "s") {
		value := timeout[0 : len(timeout)-1]
		if v, err := strconv.Atoi(value); err != nil {
			panic("The timeout " + timeout + " is not valid.")
		} else {
			return v * 1000
		}
	}

	panic("The timeout " + timeout + " is not valid.")
}

func parseMaxPayloadSize(size string) int {
	if len(size) <= 2 {
		panic("The payload size " + size + " is not valid.")
	}

	unit := size[len(size)-2:]
	value := size[0 : len(size)-2]

	if unit != "KB" && unit != "MB" && unit != "GB" {
		panic("The payload size " + size + " is not valid.")
	}

	if _, err := strconv.Atoi(value); err != nil {
		panic("The payload size " + size + " is not valid.")
	}
	u := 0
	if unit == "KB" {
		u = 1024
	} else if unit == "MB" {
		u = 1024 * 1024
	} else if unit == "BB" {
		u = 1024 * 1024 * 1024
	}

	v, _ := strconv.Atoi(value)
	return u * v
}

func getAuthorizationHandlerFor(authorizationType string) EndpointAuthorizationHandler {
	if authorizationType == "@allow_all" {
		return AllowAll()
	} else if authorizationType == "@signed_in" {
		return IsSignedIn()
	} else if authorizationType == "@not_signed_in" {
		return IsNotSignedIn()
	} else if authorizationType == "@service_call" {
		return IsServiceCall()
	} else if strings.HasPrefix(authorizationType, "authority::") {
		return HasAuthority(authorizationType)
	} else if strings.HasPrefix(authorizationType, "service_authority::") {
		return HasAuthority(authorizationType)
	}

	panic("Unsupported type of authorization: " + authorizationType)
}

func formatParamsFromBraceToColon(route string) string {
	parts := strings.Split(route, "/")
	result := make([]string, 0)

	for _, p := range parts {
		if strings.HasPrefix(p, "{") && strings.HasSuffix(p, "}") {
			result = append(result, ":"+p[1:len(p)-1])
		} else {
			result = append(result, p)
		}
	}

	resultString := strings.Join(result, "/")
	return resultString
}
