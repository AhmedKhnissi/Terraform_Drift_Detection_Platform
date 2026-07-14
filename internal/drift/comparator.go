package drift

import (
	"encoding/json"
	"fmt"

	"driftdetect/internal/model"
)

// Compare diffs expected (Terraform state) against actual (cloud) resource
// states and returns a DriftReport. Resources are matched by the combination of
// their type and cloud identifier.
func Compare(expected, actual []model.ResourceState, opts DriftOptions) model.DriftReport {
	report := model.DriftReport{
		ResourceCount: len(expected),
		Summary:       map[model.DriftType]int{},
		Items:         make([]model.DriftItem, 0),
	}

	actualByKey := make(map[string]model.ResourceState, len(actual))
	for _, a := range actual {
		actualByKey[key(a.Type, a.ID)] = a
	}
	expectedKeys := make(map[string]struct{}, len(expected))
	for _, e := range expected {
		expectedKeys[key(e.Type, e.ID)] = struct{}{}
	}

	for _, e := range expected {
		act, found := actualByKey[key(e.Type, e.ID)]
		if !found {
			report.Items = append(report.Items, model.DriftItem{
				Type:     e.Type,
				Name:     e.Name,
				ID:       e.ID,
				DriftType: model.DriftDeleted,
				Message:  "resource exists in state but was not found in the cloud (deleted or renamed)",
			})
			continue
		}

		if opts.CompareAttributes {
			for _, attr := range DriftAttributes(e.Type) {
				ev, eok := e.Attributes[attr]
				av, aok := act.Attributes[attr]
				if !eok || !aok {
					continue
				}
				if !valuesEqual(ev, av) {
					report.Items = append(report.Items, model.DriftItem{
						Type:      e.Type,
						Name:      e.Name,
						ID:        e.ID,
						DriftType: model.DriftModified,
						Attribute: attr,
						Expected:  ev,
						Actual:    av,
						Message:   fmt.Sprintf("attribute %q changed", attr),
					})
				}
			}
		}

		if opts.CompareTags {
			report.Items = append(report.Items, compareTags(e, act)...)
		}
	}

	if opts.DetectOrphans {
		for _, a := range actual {
			if _, ok := expectedKeys[key(a.Type, a.ID)]; !ok {
				report.Items = append(report.Items, model.DriftItem{
					Type:     a.Type,
					Name:     a.Name,
					ID:       a.ID,
					DriftType: model.DriftOrphaned,
					Message:  "resource exists in the cloud but is not declared in state",
				})
			}
		}
	}

	report.DriftCount = len(report.Items)
	for _, it := range report.Items {
		report.Summary[it.DriftType]++
	}
	return report
}

// compareTags emits one DriftItem per tag that was added, removed, or changed.
func compareTags(exp, act model.ResourceState) []model.DriftItem {
	var items []model.DriftItem
	for k, ev := range exp.Tags {
		av, ok := act.Tags[k]
		switch {
		case !ok:
			items = append(items, model.DriftItem{
				Type: exp.Type, Name: exp.Name, ID: exp.ID,
				DriftType: model.DriftTagChange, Attribute: k,
				Expected: ev, Actual: nil,
				Message: fmt.Sprintf("tag %q removed", k),
			})
		case av != ev:
			items = append(items, model.DriftItem{
				Type: exp.Type, Name: exp.Name, ID: exp.ID,
				DriftType: model.DriftTagChange, Attribute: k,
				Expected: ev, Actual: av,
				Message: fmt.Sprintf("tag %q changed", k),
			})
		}
	}
	for k, av := range act.Tags {
		if _, ok := exp.Tags[k]; !ok {
			items = append(items, model.DriftItem{
				Type: exp.Type, Name: exp.Name, ID: exp.ID,
				DriftType: model.DriftTagChange, Attribute: k,
				Expected: nil, Actual: av,
				Message: fmt.Sprintf("tag %q added", k),
			})
		}
	}
	return items
}

// key builds the match key for a resource (type + id).
func key(t, id string) string {
	return t + "\x00" + id
}

// valuesEqual compares two attribute values canonically via JSON, so numeric
// representations (e.g. 20 vs 20.0) and strings compare as equal.
func valuesEqual(a, b interface{}) bool {
	return valueString(a) == valueString(b)
}

func valueString(v interface{}) string {
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	return string(b)
}
