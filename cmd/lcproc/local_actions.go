package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"syscall"
)

var localName = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]*$`)

func validLocalName(name string) error {
	if !localName.MatchString(name) || isRoProgram(name) {
		return fmt.Errorf("프로세스 이름은 영문·숫자·하이픈·밑줄로 입력하세요. 운영 프로세스는 변경할 수 없습니다")
	}
	return nil
}
func localProcesses(home string) ([]string, error) {
	if home == "" {
		return nil, fmt.Errorf("FATIMA_HOME이 설정되지 않았습니다. 로컬 Fatima 환경을 설정한 뒤 다시 실행하세요")
	}
	entries, err := os.ReadDir(filepath.Join(home, "app"))
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if validLocalName(e.Name()) != nil || e.Name() == "revision" {
			continue
		}
		info, err := os.Stat(filepath.Join(home, "app", e.Name()))
		if err == nil && info.IsDir() {
			names = append(names, e.Name())
		}
	}
	return names, nil
}
func localRevisions(home, name string) ([]Revision, string, error) {
	if err := validLocalName(name); err != nil {
		return nil, "", err
	}
	root := filepath.Join(home, "app", "revision", name)
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, "", fmt.Errorf("리비전을 찾을 수 없습니다: %w", err)
	}
	var result []Revision
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		var n int
		idx := strings.LastIndex(e.Name(), "_R")
		if idx < 0 {
			continue
		}
		tag := e.Name()[idx+1:]
		if _, err := fmt.Sscanf(tag, "R%d", &n); err != nil {
			continue
		}
		r := Revision{dir: filepath.Join(root, e.Name()), revision: tag, number: n}
		if info, err := e.Info(); err == nil {
			r.createDtime = info.ModTime().Format(yyyyMMddHHmmss)
		}
		if data, err := os.ReadFile(filepath.Join(r.dir, deploymentJsonFile)); err == nil {
			var d Deployment
			if json.Unmarshal(data, &d) == nil {
				r.deployment = d.Build
			}
		}
		result = append(result, r)
	}
	sort.Sort(RevisionNumbers(result))
	current, err := filepath.EvalSymlinks(filepath.Join(home, "app", name))
	if err != nil {
		return nil, "", err
	}
	return result, current, nil
}
func ensureStopped(home, name string) error {
	b, err := os.ReadFile(filepath.Join(home, "app", name, "proc", name+".pid"))
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(b)))
	if err != nil || pid < 2 {
		return fmt.Errorf("PID 파일을 확인할 수 없습니다. 프로세스 상태를 먼저 확인하세요")
	}
	err = syscall.Kill(pid, 0)
	if err == nil || err == syscall.EPERM {
		return fmt.Errorf("%s가 실행 중입니다(PID %d). rostop으로 중지한 뒤 다시 시도하세요", name, pid)
	}
	if err != syscall.ESRCH {
		return err
	}
	return nil
}
func switchLocalRevision(home, name, dir string) error {
	revisions, _, err := localRevisions(home, name)
	if err != nil {
		return err
	}
	found := false
	for _, r := range revisions {
		if r.dir == dir {
			found = true
		}
	}
	if !found {
		return fmt.Errorf("선택한 리비전이 없어졌습니다. 목록을 새로고침하세요")
	}
	if err = ensureStopped(home, name); err != nil {
		return err
	}
	link := filepath.Join(home, "app", name)
	info, err := os.Lstat(link)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink == 0 {
		return fmt.Errorf("현재 프로세스 경로가 리비전 링크가 아닙니다")
	}
	temp, err := os.MkdirTemp(filepath.Join(home, "app"), ".lcproc-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(temp)
	target, err := filepath.Rel(filepath.Dir(link), dir)
	if err != nil {
		return err
	}
	staged := filepath.Join(temp, "link")
	if err = os.Symlink(target, staged); err != nil {
		return err
	}
	return os.Rename(staged, link)
}
func duplicateLocal(home, source, target string) (err error) {
	if err = validLocalName(source); err != nil {
		return err
	}
	if err = validLocalName(target); err != nil {
		return err
	}
	destRoot := filepath.Join(home, "app", "revision", target)
	link := filepath.Join(home, "app", target)
	if _, e := os.Lstat(link); !os.IsNotExist(e) {
		return fmt.Errorf("대상 %s가 이미 있거나 접근할 수 없습니다", target)
	}
	if err = os.MkdirAll(filepath.Dir(destRoot), 0755); err != nil {
		return err
	}
	if err = os.Mkdir(destRoot, 0755); err != nil {
		return fmt.Errorf("복제 대상 리비전 폴더를 만들 수 없습니다: %w", err)
	}
	defer func() {
		if err != nil {
			os.RemoveAll(destRoot)
		}
	}()
	dir := filepath.Join(destRoot, createRevisionTag())
	if err = os.Mkdir(dir, 0755); err != nil {
		return err
	}
	src, err := filepath.EvalSymlinks(filepath.Join(home, "app", source))
	if err != nil {
		return err
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	count := 0
	seen := map[string]bool{}
	for _, e := range entries {
		if !e.Type().IsRegular() {
			continue
		}
		selected := strings.HasPrefix(e.Name(), source)
		for _, suffix := range suffixList {
			selected = selected || strings.HasSuffix(e.Name(), suffix)
		}
		if !selected {
			continue
		}
		name := e.Name()
		if strings.HasPrefix(name, source) {
			name = target + strings.TrimPrefix(name, source)
		}
		if seen[name] {
			return fmt.Errorf("파일 이름 변경 후 충돌합니다: %s", name)
		}
		seen[name] = true
		if err = copyFile(filepath.Join(src, e.Name()), filepath.Join(dir, name)); err != nil {
			return err
		}
		count++
	}
	if count == 0 {
		return fmt.Errorf("복제할 실행 파일이나 설정 파일이 없습니다")
	}
	relative, err := filepath.Rel(filepath.Dir(link), dir)
	if err != nil {
		return err
	}
	return os.Symlink(relative, link)
}
