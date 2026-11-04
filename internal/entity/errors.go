package entity

import "errors"

var (
	ErrUserNotFound       = errors.New("user not found")
	ErrUserAlreadyExists  = errors.New("user already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidRole        = errors.New("invalid user role")

	ErrInvalidContentType       = errors.New("invalid content type")
	ErrContentTypeAlreadyExists = errors.New("content type already exists")
	ErrContentTypeNotFound      = errors.New("content type not found")

	ErrInvalidComponent       = errors.New("invalid component")
	ErrComponentAlreadyExists = errors.New("component already exists")
	ErrComponentNotFound      = errors.New("component not found")

	ErrEntryNotFound = errors.New("entry not found")
	ErrInvalidEntry  = errors.New("invalid entry")
	ErrInvalidQuery  = errors.New("invalid query")

	ErrUploadNotFound        = errors.New("upload not found")
	ErrInvalidUpload         = errors.New("invalid upload")
	ErrUploadTooLarge        = errors.New("upload too large")
	ErrStorageProviderAbsent = errors.New("storage provider not configured")

	ErrFolderNotFound      = errors.New("folder not found")
	ErrFolderAlreadyExists = errors.New("folder already exists")
	ErrFolderNotEmpty      = errors.New("folder not empty")
	ErrInvalidFolder       = errors.New("invalid folder")

	ErrInvalidProject       = errors.New("invalid project")
	ErrProjectAlreadyExists = errors.New("project already exists")
	ErrProjectNotFound      = errors.New("project not found")

	ErrInvalidAPIKey  = errors.New("invalid api key")
	ErrAPIKeyNotFound = errors.New("api key not found")
)
