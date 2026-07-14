package state

import (
	"os"
	"testing"
)

func TestParseStateFixture(t *testing.T) {
	f, err := os.Open("testdata/sample.tfstate")
	if err != nil {
		t.Fatalf("open fixture: %v", err)
	}
	defer f.Close()

	resources, err := ParseState(f)
	if err != nil {
		t.Fatalf("ParseState: %v", err)
	}

	// The data source (aws_ami) must be skipped, leaving 2 managed resources.
	if len(resources) != 2 {
		t.Fatalf("expected 2 managed resources, got %d: %+v", len(resources), resources)
	}

	byName := map[string]int{}
	for i, r := range resources {
		byName[r.Name] = i
		if r.Provider != "aws" {
			t.Errorf("resource %s: expected provider aws, got %q", r.Name, r.Provider)
		}
	}

	web := resources[byName["web"]]
	if web.Type != "aws_instance" || web.ID != "i-0abc123" {
		t.Errorf("web instance mismatch: %+v", web)
	}
	if web.Attributes["instance_type"] != "t2.micro" {
		t.Errorf("expected instance_type t2.micro, got %v", web.Attributes["instance_type"])
	}
	if web.Tags["env"] != "prod" {
		t.Errorf("expected tag env=prod, got %v", web.Tags)
	}

	data := resources[byName["data"]]
	if data.Type != "aws_s3_bucket" || data.ID != "my-data-bucket" {
		t.Errorf("data bucket mismatch: %+v", data)
	}
}
