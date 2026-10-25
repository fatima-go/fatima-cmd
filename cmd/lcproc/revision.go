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
	"time"
)

// Revision directories live in app/revision/<process>. The legacy deploy and
// lcproc dup name them <time>_R<nnn>; the juno v2 deploy names them
// v2_<operation id> and keeps a former plain application directory as
// v2_<operation id>_previous. juno's hourly cleanup recognizes revisions by the
// same rule (service/revision.go); change both together.
var (
	legacyRevisionName = regexp.MustCompile(`_R([0-9]+)$`)
	v2RevisionName     = regexp.MustCompile(`^v2_[A-Za-z0-9_-]+$`)
)

const (
	legacyRevisionTime = "2006.01.02-15.04"
	v2ManifestName     = ".fatima-v2.json"
	v2PreviousSuffix   = "_previous"
	v2LabelLength      = 12

	revisionKindLegacy   = "v1"
	revisionKindV2       = "v2"
	revisionKindPrevious = "v2 이전 폴더"
)

type v2Manifest struct {
	OperationID    string
	ArtifactSHA256 string
}

// readRevisionDir recognizes one revision folder; ok is false for other names.
func readRevisionDir(root, name string, info os.FileInfo) (r Revision, ok bool) {
	r = Revision{dir: filepath.Join(root, name), name: name, time: info.ModTime()}
	if m := legacyRevisionName.FindStringSubmatch(name); m != nil {
		r.kind = revisionKindLegacy
		r.number, _ = strconv.Atoi(m[1])
		r.revision = m[0][1:]
		if t, err := time.ParseInLocation(legacyRevisionTime, strings.TrimSuffix(name, m[0]), time.Local); err == nil {
			r.time = t
		}
	} else if v2RevisionName.MatchString(name) {
		id := strings.TrimPrefix(name, "v2_")
		r.kind = revisionKindV2
		if strings.HasSuffix(id, v2PreviousSuffix) {
			r.kind = revisionKindPrevious
			id = strings.TrimSuffix(id, v2PreviousSuffix)
		}
		r.revision = id
		if len(id) > v2LabelLength {
			r.revision = id[:v2LabelLength]
		}
		if r.kind == revisionKindPrevious {
			r.revision += v2PreviousSuffix
		}
		if r.kind == revisionKindV2 {
			manifestPath := filepath.Join(r.dir, v2ManifestName)
			if mi, err := os.Stat(manifestPath); err == nil {
				r.time = mi.ModTime()
			}
			if data, err := os.ReadFile(manifestPath); err == nil {
				var m v2Manifest
				if json.Unmarshal(data, &m) == nil {
					r.operationID, r.sha256 = m.OperationID, m.ArtifactSHA256
				}
			}
		}
	} else {
		return r, false
	}
	r.createDtime = r.time.Format(yyyyMMddHHmmss)
	if data, err := os.ReadFile(filepath.Join(r.dir, deploymentJsonFile)); err == nil {
		var d Deployment
		if json.Unmarshal(data, &d) == nil {
			r.deployment = d.Build
		}
	}
	return r, true
}

// listRevisions returns recognized revisions, newest first. Legacy revisions are
// dated by their name and v2 revisions by their manifest, falling back to the
// folder time.
func listRevisions(root string) ([]Revision, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	var result []Revision
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if r, ok := readRevisionDir(root, e.Name(), info); ok {
			result = append(result, r)
		}
	}
	sort.SliceStable(result, func(i, j int) bool {
		if !result[i].time.Equal(result[j].time) {
			return result[i].time.After(result[j].time)
		}
		if result[i].number != result[j].number {
			return result[i].number > result[j].number
		}
		return result[i].name > result[j].name
	})
	return result, nil
}

// findRevision accepts the label shown in the list (R017, o_35e6d8ad1a) or
// the full folder name, ignoring case.
func findRevision(revisions []Revision, key string) (Revision, error) {
	var found []Revision
	for _, r := range revisions {
		if strings.EqualFold(r.name, key) {
			return r, nil
		}
		if strings.EqualFold(r.revision, key) {
			found = append(found, r)
		}
	}
	switch len(found) {
	case 0:
		return Revision{}, fmt.Errorf("리비전 %s를 찾을 수 없습니다", key)
	case 1:
		return found[0], nil
	}
	return Revision{}, fmt.Errorf("리비전 %s가 여러 개입니다. 폴더 이름 전체로 지정하세요", key)
}
