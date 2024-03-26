package errs

const (
	// General error codes.
	ServerInternalError = 500  // Server internal error
	ArgsError           = 1001 // Input parameter error
	NoPermissionError   = 1002 // Insufficient permission
	DuplicateKeyError   = 1003
	RecordNotFoundError = 1004 // Record does not exist
)
