package tokens_test

import (
	"strings"
	"testing"

	"github.com/fow830/ratchet/pkg/tokens"
)

func TestValidateConfigOK(t *testing.T) {
	cfg := tokens.DefaultConfig("github.com/fow830/ratchet")
	if err := tokens.Validate(cfg); err != nil {
		t.Fatal(err)
	}
}

func TestValidateConfigMissingModule(t *testing.T) {
	cfg := tokens.DefaultConfig("")
	err := tokens.Validate(cfg)
	if err == nil || !strings.Contains(err.Error(), "module") {
		t.Fatalf("want module error, got %v", err)
	}
}

func TestValidateForbiddenEdge(t *testing.T) {
	cfg := tokens.DefaultConfig("m")
	cfg.ForbiddenEdges = []tokens.ForbiddenEdge{{From: "domain", To: "delivery"}}
	if err := tokens.Validate(cfg); err != nil {
		t.Fatal(err)
	}
}

func TestValidateContractLockMode(t *testing.T) {
	cfg := tokens.DefaultConfig("m")
	cfg.ContractLocks = []tokens.ContractLock{{Path: "x.txt", Mode: tokens.LockModeRender, RenderPackage: "m/tokens", RenderFunc: "RenderX"}}
	if err := tokens.Validate(cfg); err != nil {
		t.Fatal(err)
	}
	cfg.ContractLocks[0].Mode = "bogus"
	if err := tokens.Validate(cfg); err == nil {
		t.Fatal("expected invalid lock mode error")
	}
}
