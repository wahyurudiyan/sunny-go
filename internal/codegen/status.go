package codegen

import (
	"os"
	"path/filepath"
)

// ServiceStatus reports which generation stages have produced output on
// disk for one tracked service — the same four checks `sgo list
// services` prints, and what the web UI's service dashboard shows,
// sharing this one implementation so both surfaces agree by
// construction rather than by two hand-kept-in-sync checklists.
type ServiceStatus struct {
	Name         string `json:"name"`
	Proto        bool   `json:"proto"`
	ContractGen  bool   `json:"contractGen"`
	DomainEntity bool   `json:"domainEntity"`
	ServiceImpl  bool   `json:"serviceImpl"`
}

// Status inspects projectDir for name's generated artifacts.
func Status(projectDir, name string) ServiceStatus {
	exists := func(rel string) bool {
		_, err := os.Stat(filepath.Join(projectDir, rel))
		return err == nil
	}

	return ServiceStatus{
		Name:         name,
		Proto:        exists(filepath.Join("contract", "pb", name+".proto")),
		ContractGen:  exists(filepath.Join("contract", "gen", name)),
		DomainEntity: exists(filepath.Join("internal", "domain", name)),
		ServiceImpl:  exists(filepath.Join("internal", "application", name, "service.go")),
	}
}
