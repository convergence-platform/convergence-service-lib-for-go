package lib

import uuid2 "github.com/google/uuid"

type ManagedApiError struct {
	HttpStatusCode  int
	Code            string
	Message         string
	body            any
	bodyType        *string
	RequestId       *uuid2.UUID
	ParentRequestId *string
}

func (m *ManagedApiError) Error() string {
	return m.Message
}

func (m *ManagedApiError) SetBody(body any, bodyType string) {
	m.body = body
	m.bodyType = &bodyType
}

func ConstructManagedApiError() *ManagedApiError {
	return &ManagedApiError{}
}

func (m *ManagedApiError) HasCustomBody() bool {
	return m.bodyType != nil
}

func (m *ManagedApiError) CustomBody() (any, *string) {
	return m.body, m.bodyType
}
