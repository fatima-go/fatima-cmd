package deployui

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type legacyOutcome struct {
	State, Summary string
	Targets        int
}

var legacyReceipt = regexp.MustCompile(`^far name : .+ \([0-9]+ bytes\)\. target : ([0-9]+) juno enqueued$`)
var legacyFailure = regexp.MustCompile(`(?m)^(auth fail :|fail to |far farArtifactFile doesn't exist :|far\(.+\) doesn't support platform )`)

// legacyMain also returns exit 0 on errors. Only its final, known Jupiter
// receipt confirms request completion; it never confirms Juno installation.
func legacyResult(output string, err error) legacyOutcome {
	if err != nil {
		return legacyOutcome{State: "FAILED", Summary: "배포 요청 실패 · " + clean(err.Error())}
	}
	output = strings.TrimSpace(clean(output))
	if legacyFailure.MatchString(output) {
		return legacyOutcome{State: "FAILED", Summary: "배포 요청 실패 · 아래 오류 로그를 확인하세요."}
	}
	last := output[strings.LastIndex(output, "\n")+1:]
	if match := legacyReceipt.FindStringSubmatch(last); match != nil {
		if n, e := strconv.Atoi(match[1]); e == nil && n > 0 {
			return legacyOutcome{State: "COMPLETED", Targets: n, Summary: fmt.Sprintf("배포 요청 완료 · 대상 %d개 · Jupiter 응답 수신 완료", n)}
		}
	}
	return legacyOutcome{State: "UNCONFIRMED", Summary: "명령 종료 · 완료 응답을 확인할 수 없습니다. 아래 로그를 확인하세요."}
}
