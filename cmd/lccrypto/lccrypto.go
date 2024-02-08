package main

import (
	"fmt"
	"github.com/fatima-go/fatima-core/crypt"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"strings"
)

const (
	usageTemplate = `Usage: {{if .Runnable}}{{.UseLine}}{{end}}
{{if .HasAvailableSubCommands}}{{$cmds := .Commands}}
Available Commands:{{range $cmds}}
  {{rpad .Name .NamePadding }} {{ .Short}}{{end}}{{end}}
{{if .HasAvailableSubCommands}}{{else}}{{if .HasAvailableLocalFlags}}
Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{end}}{{if .HasAvailableSubCommands}}
"{{.CommandPath}} [command] --help" 를 타이핑하면 각 명령어의 자세한 사용법을 확인할 수 있습니다.{{end}}
`
)

var (
	availableSchemes = []string{
		crypt.SecretSchemeB64,
		crypt.SecretSchemeNative,
	}
)

func main() {
	//goland:noinspection SpellCheckingInspection
	var rootCmd = &cobra.Command{
		Use:   "lccrpyto COMMAND [OPTIONS]",
		Short: "구성 변수의 값에 암복호화 / 난독화가 필요한 경우를 지원하기 위한 툴입니다.",

		SilenceUsage:          true,
		SilenceErrors:         true,
		TraverseChildren:      true,
		DisableFlagsInUseLine: true,
		DisableSuggestions:    true,

		CompletionOptions: cobra.CompletionOptions{
			DisableDefaultCmd:   true,
			HiddenDefaultCmd:    true,
			DisableDescriptions: true,
		},

		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				_ = cmd.Help()
				return
			}

			fmt.Printf("[%s]는 해당 툴에서 지원하는 명령어가 아닙니다.\nlccrypto --help 를 타이핑 하세요.", args[0])
		},

		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}

	rootCmd.AddCommand(encryptCommand(), decryptCommand())
	rootCmd.SetUsageTemplate(usageTemplate)
	rootCmd.SetHelpCommand(&cobra.Command{
		Use:    "",
		Hidden: true,
	})

	for _, subCmd := range rootCmd.Commands() {
		subCmd.DisableFlagsInUseLine = true
		subCmd.SilenceUsage = true
	}

	err := rootCmd.Execute()
	if err != nil {
		fmt.Printf("%v\n", err)
	}
}

type options struct {
	Scheme string
}

func (o options) valid() error {
	if equalsAny(o.Scheme, availableSchemes...) {
		return nil
	}

	return fmt.Errorf("현재 지원하는 암호화/난독화 스킴은 [%s] 입니다", strings.Join(availableSchemes, "|"))
}

//goland:noinspection SpellCheckingInspection
func decryptCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dec ENCRYPTEDTEXT",
		Short: "암호화문을 복호화합니다.",
		Args:  requiresMinArguments(1),

		RunE: func(cmd *cobra.Command, args []string) error {
			encryptedText := args[0]
			plaintext := crypt.ResolveSecret(encryptedText)

			if plaintext == encryptedText {
				return fmt.Errorf("지원되지 않는 형태로 암호화되어 복호화가 실패했습니다")
			}

			fmt.Printf("변환값: %s\n", plaintext)
			return nil
		},
	}

	return cmd
}

func encryptCommand() *cobra.Command {
	opt := options{}
	cmd := &cobra.Command{
		Use:   "enc [FLAGS] PLAINTEXT",
		Short: "평문을 지정된 스킴으로 암호화 합니다.",
		Args:  requiresMinArguments(1),

		RunE: func(cmd *cobra.Command, args []string) error {
			plaintext := args[0]
			err := opt.valid()
			if err != nil {
				return err
			}

			var encrypted string
			switch opt.Scheme {
			case "b64":
				encrypted = crypt.CreateSecretBase64(plaintext)
			case "native":
				encrypted = crypt.CreateSecretNative(plaintext)
			}

			fmt.Printf("변환값: %s\n", encrypted)
			return nil
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&opt.Scheme, "scheme", "s", "b64", fmt.Sprintf("암호화 방법, %s", strings.Join(availableSchemes, "|")))

	return cmd
}

func requiresMinArguments(minArgsCount int) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) >= minArgsCount {
			return nil
		}

		return errors.Errorf("수행하는데 flags를 제외한 최소한 [%d] 개의 인자가 필요합니다.\n%s -h, --help 를 타이핑 해보세요.", minArgsCount, cmd.CommandPath())
	}
}

func equalsAny[V comparable](target V, compareValues ...V) bool {
	for _, compareValue := range compareValues {
		if target == compareValue {
			return true
		}
	}

	return false
}
