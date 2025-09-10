package utils

var ERRORS = struct {
	NOT_FOUND                    string
	INVALID_REQUEST_DATA         string
	INVALID_OTP_CODE             string
	EXPIRED_OTP_CODE             string
	INVALID_CREDENTIALS          string
	TOO_MANY_OTP_REQUESTS        string
	EMAIL_ALREADY_IN_EXISTS      string
	PHONE_ALREADY_IN_EXISTS      string
	USER_NOT_FOUND               string
	USER_INACTIVE                string
	USER_BLOCKED                 string
	INVALID_TOKEN                string
	FILE_SIZE_MAX_LIMIT_EXCEEDED string
	FILE_INVALID_FORMAT          string
	GENERIC_ERROR                string
	RECORD_NOT_FOUND             string
	FOLDER_ALREADY_EXISTS        string
	FILE_ALREADY_EXISTS          string
	INTERNAL_SERVER_ERROR        string
	SMS_OTP_PROVIDER_ERROR       string
}{
	NOT_FOUND:                    "not_found",
	INVALID_REQUEST_DATA:         "invalid_request_data",
	INVALID_OTP_CODE:             "invalid_otp_code",
	EXPIRED_OTP_CODE:             "expired_otp_code",
	INVALID_CREDENTIALS:          "invalid_credentials",
	TOO_MANY_OTP_REQUESTS:        "too_many_otp_requests",
	EMAIL_ALREADY_IN_EXISTS:      "email_already_exists",
	PHONE_ALREADY_IN_EXISTS:      "phone_already_exists",
	USER_NOT_FOUND:               "user_not_found",
	USER_INACTIVE:                "user_inactive",
	USER_BLOCKED:                 "user_blocked",
	INVALID_TOKEN:                "invalid_token",
	FILE_SIZE_MAX_LIMIT_EXCEEDED: "file_size_max_limit_exceeded",
	FILE_INVALID_FORMAT:          "file_invalid_format",
	GENERIC_ERROR:                "generic_error",
	RECORD_NOT_FOUND:             "record_not_found",
	FOLDER_ALREADY_EXISTS:        "folder_already_exists",
	FILE_ALREADY_EXISTS:          "file_already_exists",
	INTERNAL_SERVER_ERROR:        "internal_server_error",
	SMS_OTP_PROVIDER_ERROR:       "sms_otp_provider_error",
}
