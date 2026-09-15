package share

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"strings"
	"testing"
)

func TestUnsupportedRPC(t *testing.T) {
	for _, code := range []codes.Code{codes.Unimplemented, codes.Unavailable, codes.PermissionDenied} {
		original := status.Error(code, "detail")
		result := UnsupportedRPC(original, "Juno")
		if status.Code(result) != code {
			t.Fatal(result)
		}
		if code == codes.Unimplemented {
			if !strings.Contains(result.Error(), "Juno") || !strings.Contains(result.Error(), "업데이트") {
				t.Fatal(result)
			}
		} else if result != original {
			t.Fatal("changed unrelated error")
		}
	}
	if UnsupportedRPC(nil, "Juno") != nil {
		t.Fatal("changed success")
	}
}
