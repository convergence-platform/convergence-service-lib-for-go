package lib

import (
	"strconv"
)

type InfrastructureMicroService struct {
	Service *BaseConvergenceService
}

type ServiceConnectionDetailsRequest struct {
	ServiceName string `json:"service_name"`
}

type ServiceConnectionDetailsResponse struct {
	Service string `json:"service"`
	Port    int    `json:"port"`
	IP      string `json:"ip"`
	Host    string `json:"host"`
}

func (s InfrastructureMicroService) GetServiceURL(serviceName string) string {
	infrastructureUrl := s.Service.GetConfiguration("application.discovery_server_url").(string)
	request := ServiceConnectionDetailsRequest{
		ServiceName: serviceName,
	}

	jwt := CreateInternalJWT(s.Service, []string{}, true)
	response := PostRequest[ServiceConnectionDetailsResponse](infrastructureUrl, "/infrastructure/get-service-info", request, jwt, 200)

	if IsSuccessful(response) {
		body := response.Body
		return "http://" + body.Host + ":" + strconv.Itoa(body.Port)
	} else {
		panic("Unable to get the base url for service (" + serviceName + ")")
	}
}

func (e ServiceConnectionDetailsResponse) GetBodyType() string {
	return "service_connection_details"
}
