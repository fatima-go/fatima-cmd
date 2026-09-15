package share

import (
	"fmt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// UnsupportedRPC gives users an actionable message without treating transport
// failures as unsupported features or claiming a whole workflow was rolled back.
func UnsupportedRPC(err error, server string) error {
	if status.Code(err) != codes.Unimplemented {
		return err
	}
	return status.Error(codes.Unimplemented, fmt.Sprintf("%s가 요청한 기능을 지원하지 않습니다. 해당 서버를 업데이트한 뒤 다시 시도하세요. 지원되는 다른 기능은 계속 사용할 수 있습니다.", server))
}
