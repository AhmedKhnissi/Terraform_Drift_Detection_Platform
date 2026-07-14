package drift

import (
	"testing"

	"driftdetect/internal/model"
)

func expInstance() model.ResourceState {
	return model.ResourceState{
		Provider: "aws",
		Type:     "aws_instance",
		Name:     "web",
		ID:       "i-0abc123",
		Attributes: map[string]interface{}{
			"instance_type": "t2.micro",
			"ami":           "ami-12345",
			"subnet_id":     "subnet-abc",
		},
		Tags: map[string]string{"Name": "web", "env": "prod"},
	}
}

func TestNoDrift(t *testing.T) {
	exp := []model.ResourceState{expInstance()}
	act := []model.ResourceState{expInstance()}
	rep := Compare(exp, act, DriftOptions{CompareAttributes: true, CompareTags: true})
	if rep.DriftCount != 0 {
		t.Fatalf("expected no drift, got %d: %+v", rep.DriftCount, rep.Items)
	}
}

func TestDeleted(t *testing.T) {
	exp := []model.ResourceState{expInstance()}
	rep := Compare(exp, nil, DriftOptions{CompareAttributes: true, CompareTags: true})
	if rep.DriftCount != 1 || rep.Items[0].DriftType != model.DriftDeleted {
		t.Fatalf("expected 1 deletion, got %+v", rep.Items)
	}
}

func TestModifiedAttribute(t *testing.T) {
	exp := expInstance()
	act := expInstance()
	act.Attributes["instance_type"] = "t2.large"
	rep := Compare([]model.ResourceState{exp}, []model.ResourceState{act}, DriftOptions{CompareAttributes: true, CompareTags: true})
	if rep.DriftCount != 1 || rep.Items[0].DriftType != model.DriftModified {
		t.Fatalf("expected 1 modification, got %+v", rep.Items)
	}
	if rep.Items[0].Expected != "t2.micro" || rep.Items[0].Actual != "t2.large" {
		t.Fatalf("unexpected modified values: %+v", rep.Items[0])
	}
}

func TestTagChangeAndAdd(t *testing.T) {
	exp := expInstance()
	act := expInstance()
	act.Tags["env"] = "dev"     // changed
	act.Tags["owner"] = "team"  // added
	rep := Compare([]model.ResourceState{exp}, []model.ResourceState{act}, DriftOptions{CompareTags: true})
	var changed, added int
	for _, it := range rep.Items {
		if it.DriftType != model.DriftTagChange {
			t.Fatalf("expected only tag changes, got %+v", it)
		}
		if it.Attribute == "env" {
			changed++
		}
		if it.Attribute == "owner" {
			added++
		}
	}
	if changed != 1 || added != 1 {
		t.Fatalf("expected 1 changed + 1 added tag, got changed=%d added=%d (%+v)", changed, added, rep.Items)
	}
}

func TestOrphan(t *testing.T) {
	exp := []model.ResourceState{expInstance()}
	orphan := model.ResourceState{Provider: "aws", Type: "aws_s3_bucket", Name: "x", ID: "bucket-x", Tags: map[string]string{}}
	rep := Compare(exp, []model.ResourceState{expInstance(), orphan}, DriftOptions{DetectOrphans: true})
	if rep.Summary[model.DriftOrphaned] != 1 {
		t.Fatalf("expected 1 orphan, got %+v", rep.Summary)
	}
}

func TestNumericEquivalence(t *testing.T) {
	// 20 (int) vs 20.0 (float) from JSON must compare equal.
	exp := model.ResourceState{Provider: "aws", Type: "aws_db_instance", Name: "db", ID: "db1",
		Attributes: map[string]interface{}{"allocated_storage": 20}}
	act := model.ResourceState{Provider: "aws", Type: "aws_db_instance", Name: "db", ID: "db1",
		Attributes: map[string]interface{}{"allocated_storage": 20.0}}
	rep := Compare([]model.ResourceState{exp}, []model.ResourceState{act}, DriftOptions{CompareAttributes: true})
	if rep.DriftCount != 0 {
		t.Fatalf("expected numeric equivalence, got %+v", rep.Items)
	}
}
