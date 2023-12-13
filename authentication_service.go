package lib

import (
	uuid2 "github.com/google/uuid"
)

type AuthenticationMicroService struct {
	Service *BaseConvergenceService
	URL     string
}

type RegisterServiceAuthorityRequestDTO struct {
	UUID        uuid2.UUID `json:"uuid"`
	Authority   string     `json:"authority"`
	DisplayName string     `json:"display_name"`
	Tier        int        `json:"tier"`
}

func (s AuthenticationMicroService) RegisterServiceAuthority(request RegisterServiceAuthorityRequestDTO) ApiResponse[AtomicValueDTO[bool]] {
	jwt := CreateInternalJWT(s.Service, []string{}, true)
	response := PostRequest[AtomicValueDTO[bool]](s.URL, "/authentication/internal-services/register-authority", request, jwt, 200)

	if IsSuccessful(response) {
		return response
	} else {
		panic("Unable to register service authority.")
	}
}
