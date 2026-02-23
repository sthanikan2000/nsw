package model

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

// UnlockCondition represents a single condition that checks a specific dependency node's state and/or outcome.
// Both State and Outcome are optional, but at least one must be specified.
// When both are specified, they are combined with AND logic (both must match).
// Used within UnlockGroup to form AND conditions across multiple nodes.
type UnlockCondition struct {
	// NodeTemplateID references the workflow node template to check (template-level).
	// At the instance level, this is resolved to the actual workflow node ID.
	NodeTemplateID uuid.UUID `json:"nodeTemplateId"`

	// State is the expected state of the referenced node (e.g., "COMPLETED", "FAILED").
	// Optional — if nil, the node's state is not checked.
	State *string `json:"state,omitempty"`

	// Outcome is the expected outcome value of the referenced node (e.g., "APPROVED", "REJECTED").
	// Optional — if nil, the node's outcome is not checked.
	Outcome *string `json:"outcome,omitempty"`
}

// UnlockGroup represents a group of conditions that must ALL be true (AND logic).
// Multiple UnlockGroups within an UnlockConfig form OR logic.
type UnlockGroup struct {
	AllOf []UnlockCondition `json:"allOf"`
}

// UnlockConfig represents the unlock configuration for a workflow node.
// It uses Disjunctive Normal Form (DNF): an OR of AND groups.
//
// Example JSON:
//
//	{
//	  "anyOf": [
//	    {
//	      "allOf": [
//	        {"nodeTemplateId": "uuid-1", "state": "COMPLETED", "outcome": "APPROVED"},
//	        {"nodeTemplateId": "uuid-2", "state": "COMPLETED"}
//	      ]
//	    },
//	    {
//	      "allOf": [
//	        {"nodeTemplateId": "uuid-1", "outcome": "FAST_TRACKED"}
//	      ]
//	    }
//	  ]
//	}
//
// This means: (node1.state == COMPLETED && node1.outcome == APPROVED && node2.state == COMPLETED) OR (node1.outcome == FAST_TRACKED)
type UnlockConfig struct {
	AnyOf []UnlockGroup `json:"anyOf"`
}

// Validate checks that the unlock configuration is well-formed.
func (uc *UnlockConfig) Validate() error {
	if len(uc.AnyOf) == 0 {
		return fmt.Errorf("unlock configuration must have at least one group in anyOf")
	}
	for i, group := range uc.AnyOf {
		if len(group.AllOf) == 0 {
			return fmt.Errorf("unlock configuration group %d must have at least one condition in allOf", i)
		}
		for j, cond := range group.AllOf {
			if cond.NodeTemplateID == uuid.Nil {
				return fmt.Errorf("unlock configuration group %d condition %d has nil nodeTemplateId", i, j)
			}
			if cond.State == nil && cond.Outcome == nil {
				return fmt.Errorf("unlock configuration group %d condition %d must specify at least one of state or outcome", i, j)
			}
			if cond.State != nil && *cond.State == "" {
				return fmt.Errorf("unlock configuration group %d condition %d has empty state", i, j)
			}
			if cond.Outcome != nil && *cond.Outcome == "" {
				return fmt.Errorf("unlock configuration group %d condition %d has empty outcome", i, j)
			}
		}
	}
	return nil
}

// ResolveToInstanceIDs creates a copy of the UnlockConfig with template IDs replaced by instance node IDs.
// The templateToNodeID map should contain template ID -> node instance ID mappings.
func (uc *UnlockConfig) ResolveToInstanceIDs(templateToNodeID map[uuid.UUID]uuid.UUID) (*UnlockConfig, error) {
	resolved := &UnlockConfig{
		AnyOf: make([]UnlockGroup, len(uc.AnyOf)),
	}
	for i, group := range uc.AnyOf {
		resolved.AnyOf[i] = UnlockGroup{
			AllOf: make([]UnlockCondition, len(group.AllOf)),
		}
		for j, cond := range group.AllOf {
			nodeID, found := templateToNodeID[cond.NodeTemplateID]
			if !found {
				return nil, fmt.Errorf("no instance node found for template ID %s in unlock configuration", cond.NodeTemplateID)
			}
			resolved.AnyOf[i].AllOf[j] = UnlockCondition{
				NodeTemplateID: nodeID, // At instance level, this field holds the node instance ID
				State:          cond.State,
				Outcome:        cond.Outcome,
			}
		}
	}
	return resolved, nil
}

// Evaluate checks if the unlock conditions are satisfied given the current node states and outcomes.
// The nodeMap should contain node ID -> WorkflowNode mappings with current states.
func (uc *UnlockConfig) Evaluate(nodeMap map[uuid.UUID]WorkflowNode) bool {
	// DNF evaluation: any group being satisfied makes the whole config satisfied (OR)
	for _, group := range uc.AnyOf {
		if uc.evaluateGroup(group, nodeMap) {
			return true
		}
	}
	return false
}

// evaluateGroup checks if all conditions in a group are satisfied (AND).
func (uc *UnlockConfig) evaluateGroup(group UnlockGroup, nodeMap map[uuid.UUID]WorkflowNode) bool {
	for _, cond := range group.AllOf {
		node, exists := nodeMap[cond.NodeTemplateID] // At instance level, NodeTemplateID holds the node instance ID
		if !exists {
			return false
		}
		// Check state if specified
		if cond.State != nil && string(node.State) != *cond.State {
			return false
		}
		// Check outcome if specified
		if cond.Outcome != nil {
			if node.Outcome == nil || *node.Outcome != *cond.Outcome {
				return false
			}
		}
	}
	return true
}

// MarshalJSON implements json.Marshaler for UnlockConfig.
func (uc UnlockConfig) MarshalJSON() ([]byte, error) {
	type Alias UnlockConfig
	return json.Marshal(Alias(uc))
}

// UnmarshalJSON implements json.Unmarshaler for UnlockConfig.
func (uc *UnlockConfig) UnmarshalJSON(data []byte) error {
	type Alias UnlockConfig
	var alias Alias
	if err := json.Unmarshal(data, &alias); err != nil {
		return err
	}
	*uc = UnlockConfig(alias)
	return nil
}
