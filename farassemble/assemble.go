// Package farassemble gofar 가 deployment.json 에 기록한 바이너리별 설치 정책(binaries)에 따라
// 배포 대상 환경(profile)용 FAR 를 임시로 재조립한다. 원본 FAR 는 변경하지 않는다.
package farassemble

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	DeploymentJson = "deployment.json"
	PlatformDir    = "platform"
	RoleProcess    = "process"
	tempDirPrefix  = "rodeploy-"
	staleTempAge   = 24 * time.Hour
)

// BinarySpec gofar 가 기록하는 바이너리 정책
type BinarySpec struct {
	Role    string       `json:"role"`
	Strip   bool         `json:"strip,omitempty"`
	Install *InstallRule `json:"install,omitempty"`
}

type InstallRule struct {
	Default  bool            `json:"default"`
	Profiles map[string]bool `json:"profiles,omitempty"`
}

// Installed 해당 profile 에 설치되는지 판단한다. install 이 없으면 항상 설치한다
func (s BinarySpec) Installed(profile string) bool {
	if s.Install == nil {
		return true
	}
	if v, ok := s.Install.Profiles[profile]; ok {
		return v
	}
	return s.Install.Default
}

// Assembly 재조립한 FAR 의 deployment.json 에 남기는 기록
type Assembly struct {
	Profile      string   `json:"profile"`
	Removed      []string `json:"removed"`
	SourceSha256 string   `json:"source_sha256"`
}

// Deployment FAR 의 deployment.json 중 재조립에 필요한 부분
type Deployment struct {
	Process  string                `json:"process"`
	Binaries map[string]BinarySpec `json:"binaries"`
	Assembly *Assembly             `json:"assembly"`
}

// HasInstallPolicy profile 에 따라 설치 여부가 달라지는 바이너리가 있는지 확인한다
func (d Deployment) HasInstallPolicy() bool {
	for _, spec := range d.Binaries {
		if spec.Install != nil {
			return true
		}
	}
	return false
}

// Removed profile 에 설치하지 않는 바이너리 목록을 구한다
func (d Deployment) Removed(profile string) ([]string, error) {
	removed := make([]string, 0)
	for name, spec := range d.Binaries {
		if spec.Installed(profile) {
			continue
		}
		if spec.Role == RoleProcess || name == d.Process {
			return nil, fmt.Errorf("process binary %s is not installed for profile %q. check .gofar/gofar.yaml", name, profile)
		}
		removed = append(removed, name)
	}
	sort.Strings(removed)
	return removed, nil
}

// Result 재조립 결과. Assembled 가 false 이면 Path 는 원본 FAR 이다.
// HasPolicy 는 FAR 에 바이너리 정책(binaries)이 있는지를 나타낸다
type Result struct {
	Path      string
	Profile   string
	Removed   []string
	Assembled bool
	HasPolicy bool
}

// Summary 사용자에게 보여줄 재조립 요약
func (r Result) Summary() string {
	if !r.Assembled {
		return fmt.Sprintf("profile %q : 원본 FAR 를 그대로 배포합니다", r.Profile)
	}
	return fmt.Sprintf("profile %q : %s 를 제외한 FAR 를 배포합니다", r.Profile, strings.Join(r.Removed, ","))
}

// ReadDeployment FAR 안의 deployment.json 을 읽는다. 없으면 nil 을 리턴한다
func ReadDeployment(far string) (*Deployment, error) {
	z, err := zip.OpenReader(far)
	if err != nil {
		return nil, err
	}
	defer z.Close()

	for _, f := range z.File {
		if entryName(f.Name) != DeploymentJson {
			continue
		}
		data, err := readEntry(f)
		if err != nil {
			return nil, err
		}
		dep := &Deployment{}
		if err = json.Unmarshal(data, dep); err != nil {
			return nil, fmt.Errorf("invalid %s : %s", DeploymentJson, err.Error())
		}
		return dep, nil
	}
	return nil, nil
}

// ForProfile profile 에 설치하지 않는 바이너리를 제외한 FAR 를 임시 디렉토리에 만든다.
// 제외할 바이너리가 없으면 원본을 그대로 리턴한다. cleanup 은 항상 호출해야 한다
func ForProfile(far, profile string) (Result, func(), error) {
	result := Result{Path: far, Profile: profile}
	noop := func() {}

	dep, err := ReadDeployment(far)
	if err != nil {
		return result, noop, err
	}
	if dep == nil || len(dep.Binaries) == 0 {
		return result, noop, nil
	}
	result.HasPolicy = true
	if dep.Assembly != nil {
		if dep.Assembly.Profile == profile {
			// 같은 profile 로 이미 조립된 FAR 는 그대로 사용할 수 있다
			return result, noop, nil
		}
		return result, noop, fmt.Errorf("FAR is already assembled for profile %q (removed %s). deploy the original FAR built by gofar",
			dep.Assembly.Profile, strings.Join(dep.Assembly.Removed, ","))
	}

	removed, err := dep.Removed(profile)
	if err != nil {
		return result, noop, err
	}
	if len(removed) == 0 {
		return result, noop, nil
	}

	sweepStaleTempDirs()
	dir, err := os.MkdirTemp("", tempDirPrefix+safeName(dep.Process)+"-")
	if err != nil {
		return result, noop, fmt.Errorf("fail to create tmp dir : %s", err.Error())
	}
	cleanup := func() { _ = os.RemoveAll(dir) }

	target := filepath.Join(dir, filepath.Base(far))
	if err = assemble(far, target, profile, removed); err != nil {
		cleanup()
		return result, noop, err
	}

	result.Path = target
	result.Removed = removed
	result.Assembled = true
	return result, cleanup, nil
}

// assemble 원본 항목을 압축 해제 없이 그대로 복사하고 제외 대상만 건너뛴다.
// 같은 원본과 profile 이면 항상 같은 파일이 만들어진다 (재업로드시 sha256 동일)
func assemble(source, target, profile string, removed []string) error {
	sum, err := fileSha256(source)
	if err != nil {
		return err
	}

	z, err := zip.OpenReader(source)
	if err != nil {
		return err
	}
	defer z.Close()

	out, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return err
	}
	defer out.Close()

	zw := zip.NewWriter(out)
	removedSet := make(map[string]bool)
	for _, name := range removed {
		removedSet[name] = true
	}

	for _, f := range z.File {
		name := entryName(f.Name)
		if isRemovedBinary(name, f, removedSet) {
			continue
		}
		if name == DeploymentJson {
			if err = writeDeployment(zw, f, Assembly{Profile: profile, Removed: removed, SourceSha256: sum}); err != nil {
				return err
			}
			continue
		}
		if err = zw.Copy(f); err != nil {
			return fmt.Errorf("fail to copy %s : %s", f.Name, err.Error())
		}
	}

	if err = zw.Close(); err != nil {
		return err
	}
	return out.Close()
}

// isRemovedBinary platform/<os_arch>/<name> 또는 루트(플랫폼 축소된 FAR)의 제외 대상 바이너리인지 확인한다
func isRemovedBinary(name string, f *zip.File, removed map[string]bool) bool {
	if f.FileInfo().IsDir() {
		return false
	}
	parts := strings.Split(name, "/")
	switch {
	case len(parts) == 3 && parts[0] == PlatformDir:
		return removed[parts[2]]
	case len(parts) == 1:
		return removed[parts[0]]
	}
	return false
}

// writeDeployment 원본 deployment.json 에 assembly 를 추가한다. 다른 값은 그대로 유지한다
func writeDeployment(zw *zip.Writer, f *zip.File, assembly Assembly) error {
	data, err := readEntry(f)
	if err != nil {
		return err
	}
	var m map[string]json.RawMessage
	if err = json.Unmarshal(data, &m); err != nil {
		return fmt.Errorf("invalid %s : %s", DeploymentJson, err.Error())
	}
	b, err := json.Marshal(assembly)
	if err != nil {
		return err
	}
	m["assembly"] = b
	data, err = json.Marshal(m)
	if err != nil {
		return err
	}

	header := &zip.FileHeader{
		Name:           f.Name,
		Method:         f.Method,
		Modified:       f.Modified,
		CreatorVersion: f.CreatorVersion,
		ExternalAttrs:  f.ExternalAttrs,
	}
	w, err := zw.CreateHeader(header)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

// entryName gofar 는 '/' 를 루트로 하는 경로를 사용하므로 앞의 '/' 를 제거한 이름을 구한다
func entryName(name string) string {
	return strings.TrimPrefix(name, "/")
}

func readEntry(f *zip.File) ([]byte, error) {
	r, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return io.ReadAll(io.LimitReader(r, 1024*1024))
}

func fileSha256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err = io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func safeName(name string) string {
	if name == "" {
		return "far"
	}
	return strings.Map(func(r rune) rune {
		if r == '/' || r == os.PathSeparator {
			return '_'
		}
		return r
	}, name)
}

// sweepStaleTempDirs 비정상 종료로 남은 오래된 재조립 디렉토리를 정리한다
func sweepStaleTempDirs() {
	entries, err := os.ReadDir(os.TempDir())
	if err != nil {
		return
	}
	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasPrefix(entry.Name(), tempDirPrefix) {
			continue
		}
		info, err := entry.Info()
		if err != nil || time.Since(info.ModTime()) < staleTempAge {
			continue
		}
		_ = os.RemoveAll(filepath.Join(os.TempDir(), entry.Name()))
	}
}
