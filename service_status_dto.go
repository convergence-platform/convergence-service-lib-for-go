package lib

type ConvergenceEndpointRateLimitPolicy struct {
	Policy   string `json:"policy"`
	Count    int    `json:"count"`
	Duration int    `json:"duration"`
}

type ServiceEndpointInfoDTO struct {
	URL                       string                               `json:"url"`
	Method                    string                               `json:"method"`
	ExposedThroughGateway     bool                                 `json:"exposed_through_gateway"`
	AuthorizationTypeExpected string                               `json:"expected_authorization"`
	MaxPayloadSize            int                                  `json:"max_payload_size"`
	Timeout                   int                                  `json:"timeout"`
	RateLimitingPolicy        []ConvergenceEndpointRateLimitPolicy `json:"rate_limiting_policy"`
	MaintenanceMode           string                               `json:"maintenance_mode"`
	Accepts                   []string                             `json:"accepts"`
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
