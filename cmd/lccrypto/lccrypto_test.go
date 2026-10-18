package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fatima-go/fatima-core/v2/crypt"
)

func execute(args ...string) (string, string, error) {
	cmd := newRootCommand()
	var out, stderr bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&stderr)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), stderr.String(), err
}

func TestFileRoundTrip(t *testing.T) {
	for _, scheme := range []string{"native", "b64"} {
		for _, plaintext := range []string{"", "  hello \t\nworld\r\n", "한글\x00\xff"} {
			t.Run(scheme+base64.StdEncoding.EncodeToString([]byte(plaintext)), func(t *testing.T) {
				dir := t.TempDir()
				input, encrypted, restored := filepath.Join(dir, "input"), filepath.Join(dir, "encrypted"), filepath.Join(dir, "restored")
				if err := os.WriteFile(input, []byte(plaintext), 0600); err != nil {
					t.Fatal(err)
				}
				args := []string{"enc", "-f", input, "-o", encrypted}
				if scheme == "b64" {
					args = append(args, "-s", scheme)
				}
				out, stderr, err := execute(args...)
				if err != nil || out != "" || stderr != "" {
					t.Fatalf("enc: %q %q %v", out, stderr, err)
				}
				data, err := os.ReadFile(encrypted)
				if err != nil || !strings.HasPrefix(string(data), scheme+":") {
					t.Fatalf("encrypted: %q %v", data, err)
				}
				info, err := os.Stat(encrypted)
				if err != nil || info.Mode().Perm()&0077 != 0 {
					t.Fatalf("file permissions: %v %v", info, err)
				}
				out, stderr, err = execute("dec", "-f", encrypted, "-o", restored)
				if err != nil || out != "" || stderr != "" {
					t.Fatalf("dec: %q %q %v", out, stderr, err)
				}
				data, err = os.ReadFile(restored)
				if err != nil || string(data) != plaintext {
					t.Fatalf("restored: %q want %q (%v)", data, plaintext, err)
				}
			})
		}
	}
}

func TestCompatibility(t *testing.T) {
	for _, plaintext := range []string{"", "hello", "  a\nb  ", strings.Repeat("x", 64)} {
		for _, secret := range []string{crypt.CreateSecretNative(plaintext), crypt.CreateSecretBase64(plaintext)} {
			got, err := decryptSecret(secret)
			if err != nil || got != plaintext {
				t.Fatalf("round trip: %q %v", got, err)
			}
		}
	}
}

func TestInvalidCiphertexts(t *testing.T) {
	for _, value := range []string{"", "hello", "foo:bar", "aws:test", "b64:!", "b64:Zh==", "native:", "native:YQ==", "native:!"} {
		t.Run(value, func(t *testing.T) {
			if _, err := decryptSecret(value); err == nil {
				t.Fatal("expected error")
			}
		})
	}
	// A known legacy ciphertext encrypted with profile local must fail with prod.
	t.Setenv("FATIMA_PROFILE", "prod")
	if _, err := decryptSecret("native:hBpYh7uaS8Hs8faU7xU5TA=="); err == nil {
		t.Fatal("expected profile mismatch error")
	}
}

func TestCLIValidation(t *testing.T) {
	input := filepath.Join(t.TempDir(), "input")
	if err := os.WriteFile(input, []byte("b64:YQ==\nb64:Yg==\n"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"enc"}, {"dec"}, {"unknown"}, {"enc", "-s", "invalid", "hello"},
		{"enc", "--unknown"}, {"enc", "-f", ""}, {"enc", "-f", input, "hello"},
		{"dec", "-f", input, "hello"}, {"enc", "-f", input + "missing"},
		{"enc", "hello", "-o", ""}, {"dec", "b64:YQ==", "b64:Yg==", "-o", input + "out"},
		{"dec", "-f", input, "-o", input + "out"},
	} {
		if out, _, err := execute(args...); err == nil || out != "" {
			t.Fatalf("%v: %q %v", args, out, err)
		}
	}
	out, _, err := execute("dec", "-f", input)
	if err != nil || !strings.Contains(out, "[a]") || !strings.Contains(out, "[b]") {
		t.Fatalf("batch dec: %q %v", out, err)
	}
	out, _, err = execute("enc", "--help")
	if err != nil || !strings.Contains(out, "--file") || !strings.Contains(out, "--output") || !strings.Contains(out, `default "native"`) || strings.Contains(out, "lccrpyto") {
		t.Fatalf("help: %q %v", out, err)
	}
	out, _, err = execute("enc", "-s", "b64", "  hello  ")
	if err != nil || !strings.Contains(out, "b64:ICBoZWxsbyAg") {
		t.Fatalf("whitespace: %q %v", out, err)
	}
}

func TestNewOutputOnly(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "result")
	if err := os.WriteFile(path, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, command := range []string{"enc", "dec"} {
		value := "hello"
		if command == "dec" {
			value = "b64:aGVsbG8="
		}
		if out, _, err := execute(command, value, "-o", path); err == nil || out != "" {
			t.Fatalf("existing output: %q %v", out, err)
		}
	}
	data, _ := os.ReadFile(path)
	if string(data) != "keep" {
		t.Fatal("existing file modified")
	}
	link := filepath.Join(dir, "link")
	if err := os.Symlink(path, link); err != nil {
		t.Fatal(err)
	}
	if err := writeNewFile(link, []byte("replace")); err == nil {
		t.Fatal("symlink accepted")
	}
	batch := filepath.Join(dir, "batch")
	if _, _, err := execute("enc", "-s", "b64", "a", "b", "-o", batch); err != nil {
		t.Fatal(err)
	}
	data, _ = os.ReadFile(batch)
	if string(data) != "b64:YQ==\nb64:Yg==" {
		t.Fatalf("batch output: %q", data)
	}
	failed := filepath.Join(dir, "failed")
	if _, _, err := execute("dec", "native:YQ==", "-o", failed); err == nil {
		t.Fatal("bad input accepted")
	}
	if _, err := os.Stat(failed); !os.IsNotExist(err) {
		t.Fatal("file created despite failure")
	}
}

func TestProcessErrors(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "lccrypto")
	if out, err := exec.Command("go", "build", "-o", binary, ".").CombinedOutput(); err != nil {
		t.Fatalf("build: %s %v", out, err)
	}

	for _, profile := range []string{"", "LOCAL", "prod", strings.Repeat("프로파일", 10)} {
		dir := t.TempDir()
		input, encrypted, restored := filepath.Join(dir, "in"), filepath.Join(dir, "enc"), filepath.Join(dir, "out")
		plaintext := []byte("  profile test\n\x00\xff")
		if err := os.WriteFile(input, plaintext, 0600); err != nil {
			t.Fatal(err)
		}
		for _, args := range [][]string{{"enc", "--file", input, "--output", encrypted}, {"dec", "--file", encrypted, "--output", restored}} {
			cmd := exec.Command(binary, args...)
			cmd.Env = append(os.Environ(), "FATIMA_PROFILE="+profile)
			if out, err := cmd.CombinedOutput(); err != nil || len(out) != 0 {
				t.Fatalf("profile %q: %q %v", profile, out, err)
			}
		}
		got, err := os.ReadFile(restored)
		if err != nil || !bytes.Equal(got, plaintext) {
			t.Fatalf("profile %q round trip: %q %v", profile, got, err)
		}
	}

	for _, args := range [][]string{{"enc"}, {"unknown"}, {"enc", "-s", "invalid", "x"}, {"dec", "native:YQ=="}, {"dec", "foo:bar"}} {
		cmd := exec.Command(binary, args...)
		var out, stderr bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &stderr
		err := cmd.Run()
		if err == nil || out.Len() != 0 || stderr.Len() == 0 || strings.Contains(stderr.String(), "panic:") {
			t.Fatalf("%v: out=%q err=%q status=%v", args, out.String(), stderr.String(), err)
		}
	}
}

func TestInvalidNativePadding(t *testing.T) {
	t.Setenv("FATIMA_PROFILE", "local")
	key := make([]byte, 32)
	copy(key, "local")
	block, err := aes.NewCipher(key)
	if err != nil {
		t.Fatal(err)
	}
	for _, padding := range [][]byte{
		make([]byte, 16),
		bytes.Repeat([]byte{17}, 16),
		append(bytes.Repeat([]byte{1}, 15), 2),
	} {
		ciphertext := make([]byte, 16)
		cipher.NewCBCEncrypter(block, key[:16]).CryptBlocks(ciphertext, padding)
		if _, err := decryptSecret("native:" + base64.StdEncoding.EncodeToString(ciphertext)); err == nil {
			t.Fatalf("invalid padding accepted: %v", padding)
		}
	}
}

func FuzzDecryptSecret(f *testing.F) {
	for _, value := range []string{"native:", "native:YQ==", "native:hBpYh7uaS8Hs8faU7xU5TA==", "b64:", "b64:YQ==", "foo:bar"} {
		f.Add(value)
	}
	f.Fuzz(func(t *testing.T, value string) { _, _ = decryptSecret(value) })
}
