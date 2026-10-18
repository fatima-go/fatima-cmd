package main

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/fatima-go/fatima-core/v2/crypt"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/spf13/cobra"
)

func main() {
	cmd := newRootCommand()
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(cmd.ErrOrStderr(), err)
		os.Exit(1)
	}
}

func newRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "lccrypto",
		Short:             "구성 변수의 암복호화 / 난독화를 지원합니다.",
		SilenceUsage:      true,
		SilenceErrors:     true,
		CompletionOptions: cobra.CompletionOptions{DisableDefaultCmd: true},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 0 {
				return fmt.Errorf("지원하지 않는 명령입니다. lccrypto --help 를 확인하세요")
			}
			return cmd.Help()
		},
	}
	cmd.AddCommand(newCryptoCommand(true), newCryptoCommand(false))
	return cmd
}

func newCryptoCommand(encrypt bool) *cobra.Command {
	var inputPath, outputPath, scheme string
	name, valueName, short := "dec", "ENCRYPTEDTEXT", "암호화문을 복호화합니다."
	if encrypt {
		name, valueName, short = "enc", "PLAINTEXT", "평문을 암호화합니다. 기본 스킴은 native 입니다."
	}
	cmd := &cobra.Command{
		Use:   name + " [" + valueName + "...]",
		Short: short,
		Long: short + "\n공백과 개행을 보존합니다. 여러 인자를 처리할 수 있습니다.\n" +
			"-f는 위치 인자와 함께 사용할 수 없습니다. enc -f는 파일 전체를 하나의 값으로 처리합니다.\n" +
			"dec -f는 한 줄에 하나씩 저장된 암호문을 읽습니다. dec -o는 암호문 하나만 허용합니다.\n" +
			"-o는 결과만 새 파일에 저장하며 기존 파일을 덮어쓰지 않습니다.\n" +
			"native는 FATIMA_PROFILE(미설정 시 local)이 같아야 복호화할 수 있습니다.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if encrypt && scheme != crypt.SecretSchemeNative && scheme != crypt.SecretSchemeB64 {
				return fmt.Errorf("지원하는 스킴은 native|b64 입니다")
			}
			if cmd.Flags().Changed("file") {
				if inputPath == "" || len(args) != 0 {
					return fmt.Errorf("-f에는 파일 경로만 지정하고 위치 인자를 함께 사용하지 마세요")
				}
				data, err := os.ReadFile(inputPath)
				if err != nil {
					return fmt.Errorf("입력 파일 읽기 실패: %w", err)
				}
				args = []string{string(data)}
				if !encrypt {
					// Encrypted output contains one value per line; tolerate a final line ending.
					args = strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")
					for i := range args {
						args[i] = strings.TrimSuffix(args[i], "\r")
					}
				}
			}
			if len(args) == 0 {
				return fmt.Errorf("입력값 또는 -f 파일 경로가 필요합니다")
			}
			fileOutput := cmd.Flags().Changed("output")
			if fileOutput && outputPath == "" {
				return fmt.Errorf("-o에는 파일 경로가 필요합니다")
			}
			if fileOutput && !encrypt && len(args) != 1 {
				return fmt.Errorf("dec -o는 암호문 하나만 허용합니다")
			}
			results := make([]string, len(args))
			for i, value := range args {
				if encrypt {
					if scheme == crypt.SecretSchemeB64 {
						results[i] = crypt.CreateSecretBase64(value)
					} else {
						results[i] = crypt.CreateSecretNative(value)
					}
				} else {
					resolved, err := decryptSecret(value)
					if err != nil {
						return fmt.Errorf("%d번째 입력 복호화 실패: %w", i+1, err)
					}
					results[i] = resolved
				}
			}
			if fileOutput {
				return writeNewFile(outputPath, []byte(strings.Join(results, "\n")))
			}
			writer := table.NewWriter()
			if encrypt {
				writer.AppendHeader(table.Row{"PLAINTEXT", "SECRET_VALUE"})
			} else {
				writer.AppendHeader(table.Row{"SECRET_VALUE", "PLAINTEXT"})
			}
			for i, value := range args {
				if encrypt {
					writer.AppendRow(table.Row{fmt.Sprintf("[%s]", value), results[i]})
				} else {
					writer.AppendRow(table.Row{value, fmt.Sprintf("[%s]", results[i])})
				}
			}
			_, err := fmt.Fprintln(cmd.OutOrStdout(), writer.Render())
			return err
		},
	}
	cmd.Flags().StringVarP(&inputPath, "file", "f", "", "입력 파일 경로")
	cmd.Flags().StringVarP(&outputPath, "output", "o", "", "결과를 저장할 새 파일 경로 (기존 파일 사용 불가)")
	if encrypt {
		cmd.Flags().StringVarP(&scheme, "scheme", "s", crypt.SecretSchemeNative, "암호화 방법: native|b64")
	}
	return cmd
}

// decryptSecret validates errors explicitly rather than relying on ResolveSecret's
// unchanged-string fallback. Native decoding preserves fatima-core v2.0.0's format.
func decryptSecret(value string) (string, error) {
	scheme, encoded, ok := strings.Cut(value, ":")
	if !ok || (scheme != crypt.SecretSchemeNative && scheme != crypt.SecretSchemeB64) {
		return "", fmt.Errorf("암호문은 native: 또는 b64: 형식이어야 합니다")
	}
	data, err := base64.StdEncoding.Strict().DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("잘못된 Base64 형식입니다")
	}
	if scheme == crypt.SecretSchemeB64 {
		return string(data), nil
	}
	if len(data) == 0 || len(data)%aes.BlockSize != 0 {
		return "", fmt.Errorf("잘못된 native 암호문 길이입니다")
	}
	profile := os.Getenv("FATIMA_PROFILE")
	if profile == "" {
		profile = "local"
	}
	key := make([]byte, crypt.CipherKeyBytesLength)
	copy(key, []byte(strings.ToLower(profile)))
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	plaintext := make([]byte, len(data))
	cipher.NewCBCDecrypter(block, key[:aes.BlockSize]).CryptBlocks(plaintext, data)
	padding := int(plaintext[len(plaintext)-1])
	if padding < 1 || padding > aes.BlockSize || padding > len(plaintext) {
		return "", fmt.Errorf("잘못된 패딩입니다. 암호문과 FATIMA_PROFILE을 확인하세요")
	}
	for _, b := range plaintext[len(plaintext)-padding:] {
		if int(b) != padding {
			return "", fmt.Errorf("잘못된 패딩입니다. 암호문과 FATIMA_PROFILE을 확인하세요")
		}
	}
	return string(plaintext[:len(plaintext)-padding]), nil
}

func writeNewFile(path string, data []byte) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return fmt.Errorf("새 출력 파일 생성 실패: %w", err)
	}
	_, writeErr := file.Write(data)
	closeErr := file.Close()
	if writeErr != nil || closeErr != nil {
		// Never leave a partial result that looks like a successful output file.
		removeErr := os.Remove(path)
		return fmt.Errorf("출력 파일 저장 실패: %w", errors.Join(writeErr, closeErr, removeErr))
	}
	return nil
}
