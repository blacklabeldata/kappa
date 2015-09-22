package common

type StatusCode int

// Success codes
const (
	OK StatusCode = iota + 2000
	NamespaceAlreadyExists
	UserAlreadyExists
)

// Authentication related error codes
const (
	Unauthorized StatusCode = iota + 4000
)

// Error codes
const (
	InternalServerError StatusCode = iota + 5000
	InvalidStatementType
	NamespaceDoesNotExist
	UserDoesNotExist
	CreateNamespaceError
)

var statusCodes = map[StatusCode]string{

	// Success
	OK: "OK",
	NamespaceAlreadyExists: "NamespaceAlreadyExists",
	UserAlreadyExists:      "UserAlreadyExists",

	// Security errors
	Unauthorized: "Unauthorized",

	// General errors
	InternalServerError:   "InternalServerError",
	InvalidStatementType:  "InvalidStatementType",
	NamespaceDoesNotExist: "NamespaceDoesNotExist",
	UserDoesNotExist:      "UserDoesNotExist",
	CreateNamespaceError:  "CreateNamespaceError",
}
