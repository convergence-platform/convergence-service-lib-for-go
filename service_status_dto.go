package lib

type ServiceEndpointInfoDTO struct {
	URL                       string `json:"url"`
	Method                    string `json:"method"`
	ExposedThroughGateway     bool   `json:"exposed_through_gateway"`
	AuthorizationTypeExpected string `json:"expected_authorization"`
}

type ServiceStatusDTO struct {
	ServiceName string                    `json:"service_name"`
	VersionHash string                    `json:"version_hash"`
	Version     string                    `json:"version"`
	Status      string                    `json:"status"`
	Endpoints   []*ServiceEndpointInfoDTO `json:"endpoints"`
	Extra       map[string]string         `json:"extra"`
}

func (e ServiceStatusDTO) GetBodyType() string {
	return "service_status"
}
