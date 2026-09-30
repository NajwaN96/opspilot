package platform

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var nameRE = regexp.MustCompile(`^[a-z][a-z0-9-]{1,40}$`)

type Catalog struct {
	APIVersion string    `json:"apiVersion"`
	Kind       string    `json:"kind"`
	Services   []Service `json:"services"`
}

type Service struct {
	APIVersion string         `json:"apiVersion"`
	Kind       string         `json:"kind"`
	Metadata   map[string]any `json:"metadata"`
	Spec       map[string]any `json:"spec"`
}

type Runbook struct {
	ID                 string   `json:"id"`
	Title              string   `json:"title"`
	Service            string   `json:"service"`
	Symptoms           []string `json:"symptoms"`
	Detection          string   `json:"detection"`
	Evidence           []string `json:"evidence"`
	LikelyCauses       []string `json:"likelyCauses"`
	SafeInvestigation  []string `json:"safeInvestigation"`
	AllowedRemediation []string `json:"allowedRemediation"`
	Verification       []string `json:"verification"`
	Escalation         []string `json:"escalation"`
}

type Library struct {
	Runbooks []Runbook `json:"runbooks"`
}

func FindRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for range 8 {
		if _, statErr := os.Stat(filepath.Join(dir, "platform", "catalog", "catalog.json")); statErr == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", errors.New("platform catalog not found")
}

func Load(root string) (Catalog, Library, error) {
	var catalog Catalog
	var library Library
	if err := readJSON(filepath.Join(root, "platform", "catalog", "catalog.json"), &catalog); err != nil {
		return catalog, library, err
	}
	if err := readJSON(filepath.Join(root, "platform", "runbooks", "runbooks.json"), &library); err != nil {
		return catalog, library, err
	}
	return catalog, library, nil
}

func readJSON(path string, dest any) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dest); err != nil {
		return err
	}
	return nil
}

func (c Catalog) Get(name string) (Service, bool) {
	for _, service := range c.Services {
		if fmt.Sprint(service.Metadata["name"]) == name {
			return service, true
		}
	}
	return Service{}, false
}

func (l Library) Get(id string) (Runbook, bool) {
	for _, book := range l.Runbooks {
		if book.ID == id {
			return book, true
		}
	}
	return Runbook{}, false
}

func Scorecard(catalog Catalog) map[string]any {
	rows := make([]map[string]any, 0, len(catalog.Services))
	for _, service := range catalog.Services {
		spec, _ := service.Spec["checks"].(map[string]any)
		guidance, _ := service.Spec["guidance"].(map[string]any)
		rows = append(rows, map[string]any{
			"name":     service.Metadata["name"],
			"checks":   spec,
			"guidance": guidance,
			"source":   "service-contract",
		})
	}
	return map[string]any{
		"source":   "service-contract",
		"note":     "PASS, WARNING, and MISSING come from the service contract. They are not a live cluster scan and they are not a numeric score.",
		"services": rows,
	}
}

type GenerateRequest struct {
	Name     string `json:"name"`
	Runtime  string `json:"runtime"`
	Owner    string `json:"owner"`
	SLO      string `json:"slo"`
	Strategy string `json:"strategy"`
}

func Materialize(root string, req GenerateRequest, catalog Catalog) (string, error) {
	runtime := strings.ToLower(strings.TrimSpace(req.Runtime))
	if runtime != "go" && runtime != "node" && runtime != "python" {
		return "", errors.New("runtime must be go, node, or python")
	}
	name := strings.TrimSpace(req.Name)
	if !nameRE.MatchString(name) {
		return "", errors.New("name must be a short DNS label")
	}
	if _, exists := catalog.Get(name); exists {
		return "", errors.New("refusing to overwrite an existing catalog service")
	}
	strategy := strings.ToLower(strings.TrimSpace(req.Strategy))
	if strategy != "rolling" && strategy != "canary" {
		return "", errors.New("strategy must be rolling or canary")
	}
	owner := strings.TrimSpace(req.Owner)
	slo := strings.TrimSpace(req.SLO)
	if owner == "" || slo == "" {
		return "", errors.New("owner and slo are required")
	}
	if strings.Contains(owner, "/") || strings.Contains(slo, "/") {
		return "", errors.New("owner and slo must be plain text")
	}
	dest := filepath.Join(root, "generated", "services", name)
	if _, err := os.Stat(dest); err == nil {
		return "", errors.New("generated service already exists")
	}
	src := filepath.Join(root, "templates", "services", runtime)
	if err := copyTree(src, dest, map[string]string{
		"__SERVICE_NAME__": name,
		"__OWNER__":        owner,
		"__SLO__":          slo,
		"__STRATEGY__":     strategy,
		"__RUNTIME__":      runtime,
	}); err != nil {
		_ = os.RemoveAll(dest)
		return "", err
	}
	return "generated/services/" + name, nil
}

func copyTree(src, dest string, replace map[string]string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dest, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()
		body, err := io.ReadAll(io.LimitReader(file, 1<<20))
		if err != nil {
			return err
		}
		text := string(body)
		for key, value := range replace {
			text = strings.ReplaceAll(text, key, value)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return os.WriteFile(target, []byte(text), 0o644)
	})
}
