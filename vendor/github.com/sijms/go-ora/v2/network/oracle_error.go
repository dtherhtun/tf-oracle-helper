package network

import (
	"errors"
	"strconv"
)

type OracleError struct {
	ErrCode int
	ErrMsg  string
	errPos  int
}

func NewOracleError(errCode int) *OracleError {
	return &OracleError{ErrCode: errCode, errPos: -1}
}
func (err *OracleError) Error() string {
	if len(err.ErrMsg) == 0 {
		err.translate()
	}
	var output string
	if err.errPos >= 0 {
		output = err.ErrMsg + " error occur at position: " + strconv.Itoa(err.errPos)
	} else {
		output = err.ErrMsg
	}
	//return err.ErrMsg
	return output
}

// ErrPos return sql error position。
func (err *OracleError) ErrPos() int {
	return err.errPos
}

func (err *OracleError) translate() {
	switch err.ErrCode {
	case 1:
		err.ErrMsg = "ORA-00001: Unique constraint violation"
	case 900:
		err.ErrMsg = "ORA-00900: Invalid SQL statement"
	case 901:
		err.ErrMsg = "ORA-00901: Invalid CREATE command"
	case 902:
		err.ErrMsg = "ORA-00902: Invalid data type"
	case 903:
		err.ErrMsg = "ORA-00903: Invalid table name"
	case 904:
		err.ErrMsg = "ORA-00904: Invalid identifier"
	case 905:
		err.ErrMsg = "ORA-00905: Misspelled keyword"
	case 906:
		err.ErrMsg = "ORA-00906: Missing left parenthesis"
	case 907:
		err.ErrMsg = "ORA-00907: Missing right parenthesis"
	case 1001:
		err.ErrMsg = "ORA-01001: invalid cursor"
	case 1013:
		err.ErrMsg = "ORA-01013: user requested cancel of current operation"
	case 12631:
		err.ErrMsg = "ORA-12631: Username retrieval failed"
	case 12564:
		err.ErrMsg = "ORA-12564: TNS connection refused"
	case 12506:
		err.ErrMsg = "ORA-12506: TNS:listener rejected connection based on service ACL filtering"
	case 12514:
		err.ErrMsg = "ORA-12514: TNS:listener does not currently know of service requested in connect descriptor"
	case 12516:
		err.ErrMsg = "ORA-12516: TNS:listener could not find available handler with matching protocol stack"
	case 3135:
		err.ErrMsg = "ORA-03135: connection lost contact"
	case 28041:
		err.ErrMsg = "ORA-28041: Authentication protocol internal error"
	default:
		err.ErrMsg = "ORA-" + strconv.Itoa(err.ErrCode)
	}
}

func (err *OracleError) Bad() bool {
	switch err.ErrCode {
	case 28, 1001, 1012, 1033, 1034, 1089, 3113, 3114, 3135, 12528, 12537:
		return true
	default:
		return false
	}
}

var ErrConnReset = errors.New("connection break due to context timeout")
