package main

import (
	"fmt"
	"time"

	"github.com/fatima-go/fatima-cmd/farassemble"
	"github.com/fatima-go/fatima-cmd/share"
)

// assembleForProfile jupiter 의 profile 에 맞게 far 를 재조립한다.
// 재조립이 필요 없거나 jupiter 가 profile 을 알려주지 않으면 원본 far 를 리턴한다
func assembleForProfile(flags share.FatimaCmdFlags, far string) (string, func(), error) {
	noop := func() {}
	if !flags.ProfileKnown {
		dep, err := farassemble.ReadDeployment(far)
		if err == nil && dep != nil && dep.HasInstallPolicy() {
			fmt.Printf("%s jupiter doesn't report profile. deploy far as it is (profile install policy is not applied)\n",
				time.Now().Format(yyyyMMddHHmmss))
		}
		return far, noop, nil
	}

	result, cleanup, err := farassemble.ForProfile(far, flags.Profile)
	if err != nil {
		return far, noop, err
	}
	if result.HasPolicy {
		fmt.Printf("%s %s\n", time.Now().Format(yyyyMMddHHmmss), result.Summary())
	}
	return result.Path, cleanup, nil
}
