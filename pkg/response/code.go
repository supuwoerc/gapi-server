package response

import "fmt"

//go:generate stringer -type=StatusCode -linecomment -output code_string.go
type StatusCode int

func (s StatusCode) Error() string {
	return fmt.Sprintf("%d", s)
}

const (
	Ok            StatusCode = 10000 // ok
	Error         StatusCode = 10001 // error
	InvalidParams StatusCode = 10002 // invalidParams
	InvalidToken  StatusCode = 10003 // invalidToken
	CancelRequest StatusCode = 10004 // cancelRequest
	RecoveryError StatusCode = 10005 // recoveryError
	InternalError StatusCode = 10006 // internalError
	TimeoutErr    StatusCode = 10007 // timeoutErr
	Busy          StatusCode = 10008 // busy
)
