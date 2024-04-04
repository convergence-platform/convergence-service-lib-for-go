package lib

func LogErrorCreateInternalErrorResponse(requestLog *RequestLog, logMessage string) error {
	requestLog.Error(logMessage)
	return &ManagedApiError{
		HttpStatusCode:  500,
		Code:            API_INTERNAL_ERROR,
		Message:         "An unexpected error occurred while processing the request, please see logs for more info.",
		RequestId:       requestLog.GetRawRequestID(),
		ParentRequestId: requestLog.ParentRequestIdentifier,
	}
}

func LogErrorCreateUnprocessableErrorResponse(requestLog *RequestLog, logMessage string) error {
	requestLog.Error(logMessage)
	return &ManagedApiError{
		HttpStatusCode:  422,
		Code:            API_INVALID_ENTITY_STATE,
		Message:         logMessage,
		RequestId:       requestLog.GetRawRequestID(),
		ParentRequestId: requestLog.ParentRequestIdentifier,
	}
}

func LogErrorCreateBadRequestResponse(requestLog *RequestLog, logMessage string) error {
	requestLog.Error(logMessage)
	return &ManagedApiError{
		HttpStatusCode:  400,
		Code:            INVALID_DATA,
		Message:         logMessage,
		RequestId:       requestLog.GetRawRequestID(),
		ParentRequestId: requestLog.ParentRequestIdentifier,
	}
}
