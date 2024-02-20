package main

import (
	"fmt"
	"github.com/fatima-go/fatima-core/crypt"
	"github.com/jedib0t/go-pretty/v6/table"
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
'{{.CommandPath}} COMMAND --help' 를 타이핑하면 각 명령어의 자세한 사용법을 확인할 수 있습니다.{{end}}
`
	plaintextFormat = "[%s]"
)

var (
	handlers = cryptoHandlers{
		&cryptoHandler{
			scheme:  crypt.SecretSchemeB64, // flags 미입력 시 첫번째 handler scheme 을 기본으로 사용
			encrypt: crypt.CreateSecretBase64,
		},
		&cryptoHandler{
			scheme:  crypt.SecretSchemeNative,
			encrypt: crypt.CreateSecretNative,
		},
	}
)

type cryptoHandler struct {
	scheme  string
	encrypt func(plaintext string) string
}

func (c *cryptoHandler) createSecret(plaintext string) string {
	return c.encrypt(plaintext)
}

type cryptoHandlers []*cryptoHandler

func (c cryptoHandlers) getSchemes() []string {
	schemes := make([]string, 0, len(c))
	for _, h := range c {
		schemes = append(schemes, h.scheme)
	}
	return schemes
}

func (c cryptoHandlers) getHandler(scheme string) *cryptoHandler {
	for _, h := range c {
		if scheme == h.scheme {
			return h
		}
	}
	return nil
}

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
	allSchemes := handlers.getSchemes()
	if equalsAnyString(o.Scheme, allSchemes...) {
		return nil
	}
	return fmt.Errorf("현재 지원하는 암호화/난독화 스킴은 [%s] 입니다", strings.Join(allSchemes, "|"))
}

//goland:noinspection SpellCheckingInspection
func decryptCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dec ENCRYPTEDTEXT",
		Short: "암호화문을 복호화합니다.",
		Long: "암호화문을 복호화합니다.\n" +
			"여러 암호화문을 동시에 복호화 할 경우 공백을 기준으로 입력하면 됩니다.\n" +
			"결과 값의 평문(PLAINTEXT)은 공백 등의 구분을 위해 [] 감싸서 출력됩니다.",
		Args: requiresMinArguments(1),

		RunE: func(cmd *cobra.Command, args []string) error {
			tableWriter := table.NewWriter()
			tableWriter.AppendHeader(table.Row{"SECRET_VALUE", "PLAINTEXT"})
			for _, secretValue := range args {
				resolved := crypt.ResolveSecret(secretValue)
				if resolved == secretValue[strings.Index(secretValue, ":")+1:] {
					resolved = "잘못된 형태의 암호화문으로 복호화 실패"
				}
				tableWriter.AppendRows([]table.Row{
					{secretValue, fmt.Sprintf(plaintextFormat, resolved)},
				})
			}

			fmt.Println(tableWriter.Render())
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
		Long: "평문을 지정된 스킴으로 암호화 합니다.\n" +
			"여러 평문을 동시에 암호화 할 경우 공백을 기준으로 입력하면 되고 공백, 특수문자 등이 포함된 경우 경우 \"\" 를 감싸면 됩니다.\n" +
			"결과 값의 평문(PLAINTEXT)은 공백 등의 구분을 위해 [] 감싸서 출력됩니다.",
		Args: requiresMinArguments(1),

		RunE: func(cmd *cobra.Command, args []string) error {
			err := opt.valid()
			if err != nil {
				return err
			}

			handler := handlers.getHandler(opt.Scheme)
			tableWriter := table.NewWriter()
			tableWriter.AppendHeader(table.Row{"PLAINTEXT", "SECRET_VALUE"})
			for _, plaintext := range args {
				trimmed := strings.TrimSpace(plaintext)
				tableWriter.AppendRows([]table.Row{
					{fmt.Sprintf(plaintextFormat, trimmed), handler.createSecret(trimmed)},
				})
			}

			fmt.Println(tableWriter.Render())
			return nil
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&opt.Scheme, "scheme", "s", handlers[0].scheme,
		fmt.Sprintf("암호화 방법, %s", strings.Join(handlers.getSchemes(), "|")))

	return cmd
}

func requiresMinArguments(minArgsCount int) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) >= minArgsCount {
			return nil
		}

		return errors.Errorf("수행하는데 flags를 제외한 최소한 [%d] 개의 인자가 필요합니다.\n%s -h, --help 를 타이핑 해보세요.",
			minArgsCount, cmd.CommandPath())
	}
}

func equalsAnyString(target string, compareValues ...string) bool {
	for _, compareValue := range compareValues {
		if target == compareValue {
			return true
		}
	}

	return false
}
