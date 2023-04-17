/*
 * Copyright 2023 github.com/fatima-go
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * @project fatima-core
 * @author jin
 * @date 23. 4. 14. 오후 5:07
 */

package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/fatima-go/fatima-cmd/juno"
	"github.com/fatima-go/fatima-cmd/share"
	"os"
)

var usage = `usage: %s [option]

display registed cron job list and executing in package

positional arguments:

optional arguments:
  -d    Debug mode
  -p string
        Host and Package. e.g) localhost:default
`

func main() {
	flag.Usage = func() {
		fmt.Printf(string(usage), os.Args[0])
	}

	fatimaFlags, err := share.BuildFatimaCmdFlags()
	if err != nil {
		fmt.Printf("fail to parse : %s", err.Error())
		return
	}

	flag.Parse()

	if len(flag.Args()) > 0 {
		flag.Usage()
		return
	}

	err = share.GetJunoEndpoint(&fatimaFlags)
	if err != nil {
		fmt.Printf("endpoint retrieve fail : %s\n", err.Error())
		return
	}

	cronCommands, err := juno.ListCronCommands(fatimaFlags)
	if err != nil {
		fmt.Printf("fail to get cron command list : %s\n", err.Error())
	}

	if len(cronCommands.Commands) == 0 {
		fmt.Printf("there is no cron command\n")
		return
	}

	interact(cronCommands)

	err = share.GetJunoEndpoint(&fatimaFlags)
	if err != nil {
		fmt.Printf("endpoint retrieve fail : %s\n", err.Error())
		return
	}

	err = juno.RerunCronCommands(fatimaFlags, userProc, userJob, userArgs)
	if err != nil {
		fmt.Printf("fail to rerun cron : %s\n", err.Error())
		return
	}

	return
}

var (
	userProc string
	userJob  string
	userArgs string
)

func interact(cronCommands juno.FatimaCronCommands) bool {
	for true {
		userEnter := interactProcessList(cronCommands)
		if userEnter < 1 || userEnter > len(cronCommands.Commands) {
			continue
		}

		cmd := cronCommands.Commands[userEnter-1]
		userProc = cmd.Process
		userEnter = interactCronCommand(cmd)
		if userEnter < 1 || userEnter > len(cmd.Jobs) {
			continue
		}
		job := cmd.Jobs[userEnter-1]
		userJob = job.Name
		if len(job.Sample) > 0 {
			userArgs = interactJobArgs(job)
		}

		fmt.Printf("%s, %s, %s\n", userProc, userJob, userArgs)
		break
	}

	return true
}

func interactProcessList(cronCommands juno.FatimaCronCommands) int {
	fmt.Printf("================\n")
	fmt.Printf("Cronjob rerun program\n\nselect process...\n")
	procIdx := 1
	for _, v := range cronCommands.Commands {
		fmt.Printf("[%d] %s\n", procIdx, v.Process)
		procIdx = procIdx + 1
	}

	fmt.Printf("================\n")
	fmt.Printf("Enter process number : ")
	var userEnter int
	fmt.Scanf("%d", &userEnter)
	return userEnter
}

func interactCronCommand(command juno.CronCommand) int {
	fmt.Printf("-------------\n")
	fmt.Printf("select job...\n")
	procIdx := 1
	for _, v := range command.Jobs {
		if len(v.Desc) > 0 {
			fmt.Printf("[%d] %s : %s\n", procIdx, v.Name, v.Desc)
		} else {
			fmt.Printf("[%d] %s\n", procIdx, v.Name)
		}

		procIdx = procIdx + 1
	}

	fmt.Printf("-------------\n")
	fmt.Printf("Enter job number : ")
	var userEnter int
	fmt.Scanf("%d", &userEnter)
	return userEnter
}

func interactJobArgs(job juno.CronJob) string {
	if len(job.Desc) > 0 {
		fmt.Printf("executing [%s] : %s\n", job.Name, job.Desc)
	} else {
		fmt.Printf("executing [%s]\n", job.Name)
	}

	fmt.Printf("argument sample : %s\n", job.Sample)
	fmt.Printf("type argument : ")
	var args string
	reader := bufio.NewReader(os.Stdin)
	args, _ = reader.ReadString('\n')
	return args
}
