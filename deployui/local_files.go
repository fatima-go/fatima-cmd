package deployui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/fatima-go/fatima-core/opm/artifact"
)

type localFAR struct {
	Path     string
	Size     int64
	Modified time.Time
}

type localFiles struct {
	Root    string
	Files   []localFAR
	Warning string
}

type farPreview struct {
	File localFAR
	Info *artifact.Info
	Err  error
}

func scanLocalFARs(gopath string) localFiles {
	// gofar intentionally writes to the first GOPATH even when sources are in
	// another workspace. Match that convention rather than scanning the PC.
	first := strings.SplitN(gopath, string(os.PathListSeparator), 2)[0]
	if first == "" {
		return localFiles{Warning: "GOPATH가 없습니다. p를 눌러 FAR 경로를 직접 입력하세요."}
	}
	r := localFiles{Root: filepath.Join(first, "far")}
	entries, e := os.ReadDir(r.Root)
	if e != nil {
		r.Warning = "로컬 FAR 목록을 읽을 수 없습니다. 경로를 직접 입력할 수 있습니다."
		return r
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		folder := filepath.Join(r.Root, entry.Name())
		files, e := os.ReadDir(folder)
		if e != nil {
			r.Warning = "일부 FAR 폴더를 읽을 수 없습니다. 읽을 수 있는 파일을 표시합니다."
			continue
		}
		for _, f := range files {
			if filepath.Ext(f.Name()) != ".far" {
				continue
			}
			info, e := f.Info()
			if e == nil && info.Mode().IsRegular() {
				r.Files = append(r.Files, localFAR{Path: filepath.Join(folder, f.Name()), Size: info.Size(), Modified: info.ModTime()})
			}
		}
	}
	sort.Slice(r.Files, func(i, j int) bool {
		if r.Files[i].Modified.Equal(r.Files[j].Modified) {
			return r.Files[i].Path < r.Files[j].Path
		}
		return r.Files[i].Modified.After(r.Files[j].Modified)
	})
	return r
}

func normalizeFARPath(path string) (string, error) {
	path = strings.TrimSpace(path)
	if len(path) >= 2 && ((path[0] == '"' && path[len(path)-1] == '"') || (path[0] == '\'' && path[len(path)-1] == '\'')) {
		path = path[1 : len(path)-1]
	}
	if path == "" {
		return "", fmt.Errorf("FAR 경로를 입력하세요.")
	}
	if strings.HasPrefix(path, "~/") {
		home, e := os.UserHomeDir()
		if e != nil {
			return "", e
		}
		path = filepath.Join(home, path[2:])
	}
	return filepath.Abs(path)
}

func inspectLocalFAR(path string) farPreview {
	p := farPreview{}
	p.File.Path, p.Err = normalizeFARPath(path)
	if p.Err != nil {
		return p
	}
	st, e := os.Stat(p.File.Path)
	if e != nil {
		p.Err = fmt.Errorf("파일을 읽을 수 없습니다: %w", e)
		return p
	}
	if !st.Mode().IsRegular() || st.Size() <= 0 || st.Size() > artifact.MaxSize {
		p.Err = fmt.Errorf("1 byte 이상 1 GiB 이하의 일반 FAR 파일을 선택하세요.")
		return p
	}
	p.File.Size, p.File.Modified = st.Size(), st.ModTime()
	p.Info, p.Err = artifact.Inspect(p.File.Path)
	return p
}
