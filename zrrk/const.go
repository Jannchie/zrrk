package zrrk

const (
	WS_OP_HEARTBEAT                  = 2
	WS_OP_HEARTBEAT_REPLY            = 3
	WS_OP_MESSAGE                    = 5
	WS_OP_USER_AUTHENTICATION        = 7
	WS_OP_CONNECT_SUCCESS            = 8
	WS_PACKAGE_HEADER_TOTAL_LENGTH   = 16
	WS_PACKAGE_OFFSET                = 0
	WS_HEADER_OFFSET                 = 4
	WS_VERSION_OFFSET                = 6
	WS_OPERATION_OFFSET              = 8
	WS_SEQUENCE_OFFSET               = 12
	WS_BODY_PROTOCOL_VERSION_NORMAL  = 0
	WS_BODY_PROTOCOL_VERSION_DEFLATE = 2
	WS_HEADER_DEFAULT_VERSION        = 1
	WS_HEADER_DEFAULT_OPERATION      = 1
	WS_HEADER_DEFAULT_SEQUENCE       = 1
	WS_AUTH_OK                       = 0
	WS_AUTH_TOKEN_ERROR              = -101
)
