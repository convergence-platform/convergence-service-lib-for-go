package lib

type ServiceEndpointDTO struct {
	URL     string   `json:"url"`
	Methods []string `json:"methods"`
}

type ServiceStatusDTO struct {
	ServiceName       string                `json:"service_name"`
	VersionHash       string                `json:"version_hash"`
	Version           string                `json:"version"`
	Status            string                `json:"status"`
	InternalEndpoints []*ServiceEndpointDTO `json:"internal_endpoints"`
	PublicEndpoints   []*ServiceEndpointDTO `json:"public_endpoints"`
	Extra             map[string]string     `json:"extra"`
}

func (e ServiceStatusDTO) GetBodyType() string {
	return "service_status"
}
