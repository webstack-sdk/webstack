package types

import (
	"fmt"
)

func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params:             DefaultParams(),
		NodeTypes:          []NodeType{},
		Nodes:              []Node{},
		ActivationKeys:     []ActivationKey{},
		NodeStatusCounters: []NodeStatusCounter{},
	}
}

// Validate checks the genesis state's structural invariants. Only invariants
// the msg handlers maintain unconditionally are enforced; param-dependent
// limits (e.g. max_activation_keys) are not, because params may be lowered by
// governance after state accrued under higher ones and export/import must
// round-trip.
func (gs GenesisState) Validate() error {
	if err := gs.Params.Validate(); err != nil {
		return fmt.Errorf("params: %w", err)
	}

	// Node types are validated first: the node loop below cross-references
	// them, which is only meaningful once the set itself is known good.
	//
	// license_type_id is checked for shape only. Whether it names a real
	// license type, and whether creator actually created it, are facts about
	// x/license state and are invisible from here; the msg handler is what
	// enforces them at registration time.
	nodeTypeIDs := make(map[string]struct{}, len(gs.NodeTypes))
	for _, nt := range gs.NodeTypes {
		if _, dup := nodeTypeIDs[nt.Id]; dup {
			return fmt.Errorf("duplicate node type %s", nt.Id)
		}
		if nt.Id == "" {
			return fmt.Errorf("node type id must not be empty")
		}
		if err := ValidateCanonicalAddress("creator", nt.Creator); err != nil {
			return fmt.Errorf("node type %s: %w", nt.Id, err)
		}
		if nt.LicenseTypeId == "" {
			return fmt.Errorf("node type %s: license_type_id must not be empty", nt.Id)
		}
		nodeTypeIDs[nt.Id] = struct{}{}
	}

	keyOperators := make(map[string]string, len(gs.ActivationKeys))
	for _, key := range gs.ActivationKeys {
		if _, dup := keyOperators[key.Address]; dup {
			return fmt.Errorf("duplicate activation key %s", key.Address)
		}
		// Canonical form, not merely decodable: these strings are the store
		// keys, so two encodings of one account would be two identities.
		if err := ValidateCanonicalAddress("activation", key.Address); err != nil {
			return fmt.Errorf("activation key %s: %w", key.Address, err)
		}
		if err := ValidateCanonicalAddress("operator", key.Operator); err != nil {
			return fmt.Errorf("activation key %s: %w", key.Address, err)
		}
		if key.Address == key.Operator {
			return fmt.Errorf("activation key %s: address equals its operator", key.Address)
		}
		if key.Status != KeyActive && key.Status != KeyDisabled {
			return fmt.Errorf("activation key %s: invalid status %q", key.Address, key.Status.String())
		}
		if key.CreatedAt.IsZero() {
			return fmt.Errorf("activation key %s: created_at must be set", key.Address)
		}
		keyOperators[key.Address] = key.Operator
	}

	nodeAddrs := make(map[string]struct{}, len(gs.Nodes))
	for _, node := range gs.Nodes {
		if _, dup := nodeAddrs[node.Address]; dup {
			return fmt.Errorf("duplicate node %s", node.Address)
		}
		if err := ValidateCanonicalAddress("node", node.Address); err != nil {
			return fmt.Errorf("node %s: %w", node.Address, err)
		}
		if err := ValidateCanonicalAddress("operator", node.Operator); err != nil {
			return fmt.Errorf("node %s: %w", node.Address, err)
		}
		if node.Status != NodeActive && node.Status != NodeDeactivated {
			return fmt.Errorf("node %s: invalid status %q", node.Address, node.Status.String())
		}
		// Referential, not merely non-empty: activation now resolves the type
		// through the registry, and node type records are never removed, so
		// every node's type is registered by construction.
		if _, exists := nodeTypeIDs[node.Type]; !exists {
			return fmt.Errorf("node %s: type %q is not a listed node type", node.Address, node.Type)
		}
		if node.LastActiveTime.IsZero() {
			return fmt.Errorf("node %s: last_active_time must be set", node.Address)
		}
		keyOperator, found := keyOperators[node.ActivatedBy]
		if !found {
			return fmt.Errorf("node %s: activated_by %q is not a listed activation key", node.Address, node.ActivatedBy)
		}
		if keyOperator != node.Operator {
			return fmt.Errorf("node %s: activated_by %q is bound to operator %s, not %s", node.Address, node.ActivatedBy, keyOperator, node.Operator)
		}
		nodeAddrs[node.Address] = struct{}{}
	}

	counterNodes := make(map[string]struct{}, len(gs.NodeStatusCounters))
	for _, nsc := range gs.NodeStatusCounters {
		if _, dup := counterNodes[nsc.NodeAddress]; dup {
			return fmt.Errorf("duplicate node status counter for %s", nsc.NodeAddress)
		}
		if _, exists := nodeAddrs[nsc.NodeAddress]; !exists {
			return fmt.Errorf("node status counter references unknown node %s", nsc.NodeAddress)
		}
		if nsc.Counter.LatestTime.IsZero() {
			return fmt.Errorf("node status counter for %s: latest_time must be set", nsc.NodeAddress)
		}
		counterNodes[nsc.NodeAddress] = struct{}{}
	}

	return nil
}
