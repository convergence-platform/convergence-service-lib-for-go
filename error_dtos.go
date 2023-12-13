package lib

// body type: request_error_info
type RequestValidationFailureDTO struct {
	Errors []*RequestValidationFieldFailureDTO `json:"errors"`
}

type RequestValidationFieldFailureDTO struct {
	Field    string   `json:"field"`
	Location string   `json:"location"`
	Messages []string `json:"error_messages"`
}
