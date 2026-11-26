package brokerpool

import (
	"fmt"
	"sort"
)

// Token rules Britive enforces on a pool. A pool can have at most MaxTokens tokens, at most
// MaxActiveTokens of them ACTIVE, and always at least one ACTIVE token: deleting the last
// active token fails with E1001 and setting it INACTIVE fails with E1003.
const (
	MaxTokens       = 3
	MaxActiveTokens = 2
)

// TokenSpec is the part of a token the provider manages.
type TokenSpec struct {
	Status      string
	Description string
}

// TokenOp is one API call that changes a pool's tokens.
type TokenOp int

const (
	// TokenOpDelete deletes a token (DELETE).
	TokenOpDelete TokenOp = iota
	// TokenOpCreate creates a token (POST). Britive creates it INACTIVE.
	TokenOpCreate
	// TokenOpUpdate changes a token's status and/or description (PATCH).
	TokenOpUpdate
)

// TokenStep is the next API call needed to bring a pool's tokens to the configured set.
// For TokenOpCreate, Description is the description to send. For TokenOpUpdate, Status and
// Description are nil when that field is not changed.
type TokenStep struct {
	Op          TokenOp
	Name        string
	Status      *string
	Description *string
}

// Describe returns the step in plain words for error messages, for example
// `set token "default" to INACTIVE`.
func (s TokenStep) Describe() string {
	switch {
	case s.Op == TokenOpDelete:
		return fmt.Sprintf("delete token %q", s.Name)
	case s.Op == TokenOpCreate:
		return fmt.Sprintf("create token %q", s.Name)
	case s.Status != nil:
		return fmt.Sprintf("set token %q to %s", s.Name, *s.Status)
	default:
		return fmt.Sprintf("update the description of token %q", s.Name)
	}
}

func (s TokenStep) String() string {
	switch s.Op {
	case TokenOpDelete:
		return "delete " + s.Name
	case TokenOpCreate:
		return "create " + s.Name
	default:
		out := "update " + s.Name
		if s.Status != nil {
			out += " status=" + *s.Status
		}
		if s.Description != nil {
			out += fmt.Sprintf(" description=%q", *s.Description)
		}
		return out
	}
}

// NextTokenStep returns the next API call that moves the pool's current tokens toward desired,
// or false when they already match. Each call it returns is one Britive accepts, given the
// rules above, and repeating it reaches desired for any valid desired set (at most 3 tokens,
// at most 2 ACTIVE, at least 1 ACTIVE). It takes the first step that applies:
//
//  1. Delete an INACTIVE token that is not in desired.
//  2. Delete an ACTIVE token that is not in desired, only while another token is ACTIVE.
//  3. Create a missing token. Britive creates it INACTIVE.
//  4. Set a token INACTIVE, only while another token is ACTIVE.
//  5. Set a token ACTIVE, only while fewer than 2 tokens are ACTIVE.
//  6. Update a description.
//
// So a new token is ACTIVE before the token it replaces is removed. Steps 4 and 5 also send
// the description when it differs, to save a call.
//
// One case needs an extra step: the pool has 3 tokens, its only ACTIVE token must be deleted,
// and the other two must stay INACTIVE. None of the steps above applies, so one of the two is
// set ACTIVE for a short time. Step 2 then deletes the old token, step 3 creates the new one,
// step 5 activates it, and step 4 sets the first one back to INACTIVE.
func NextTokenStep(current, desired map[string]TokenSpec) (TokenStep, bool) {
	names := sortedNames(current)
	active := countActive(current)

	// 1. Delete an INACTIVE token that is not in desired.
	for _, name := range names {
		if _, ok := desired[name]; !ok && current[name].Status != TokenStatusActive {
			return TokenStep{Op: TokenOpDelete, Name: name}, true
		}
	}

	// 2. Delete an ACTIVE token that is not in desired, while another token is ACTIVE.
	if active > 1 {
		for _, name := range names {
			if _, ok := desired[name]; !ok {
				return TokenStep{Op: TokenOpDelete, Name: name}, true
			}
		}
	}

	// 3. Create a missing token.
	if len(current) < MaxTokens {
		for _, name := range sortedNames(desired) {
			if _, ok := current[name]; !ok {
				description := desired[name].Description
				return TokenStep{Op: TokenOpCreate, Name: name, Description: &description}, true
			}
		}
	}

	// 4. Set a token INACTIVE, while another token is ACTIVE.
	if active > 1 {
		for _, name := range names {
			want, ok := desired[name]
			if ok && want.Status == TokenStatusInactive && current[name].Status == TokenStatusActive {
				return statusStep(name, current[name], want), true
			}
		}
	}

	// 5. Set a token ACTIVE, while fewer than 2 tokens are ACTIVE.
	if active < MaxActiveTokens {
		for _, name := range names {
			want, ok := desired[name]
			if ok && want.Status == TokenStatusActive && current[name].Status != TokenStatusActive {
				return statusStep(name, current[name], want), true
			}
		}
	}

	// 6. Update a description.
	for _, name := range names {
		want, ok := desired[name]
		if ok && want.Description != current[name].Description {
			description := want.Description
			return TokenStep{Op: TokenOpUpdate, Name: name, Description: &description}, true
		}
	}

	if TokensMatch(current, desired) {
		return TokenStep{}, false
	}

	// The extra step: no rule applies, so the only ACTIVE token must be deleted, the pool is
	// full, and the tokens that stay are INACTIVE. Activate one of them for a short time; step
	// 4 sets it back to INACTIVE once the new token is ACTIVE. This only applies when desired
	// is a set Britive allows; otherwise the temporary token could never be set back.
	if active < MaxActiveTokens && tokenSetAllowed(desired) {
		for _, name := range names {
			if _, ok := desired[name]; ok && current[name].Status != TokenStatusActive {
				status := TokenStatusActive
				return TokenStep{Op: TokenOpUpdate, Name: name, Status: &status}, true
			}
		}
	}

	// desired is a set Britive does not allow (no ACTIVE token, more than 3 tokens or more than
	// 2 ACTIVE). Send the next change as it is, so Britive rejects it with its own error.
	return directTokenStep(current, desired), true
}

// directTokenStep returns the first change between current and desired without checking
// Britive's rules: delete a token that is not in desired, create a missing one, or update one
// that differs. It is used only when desired breaks the rules, so Britive returns its error.
func directTokenStep(current, desired map[string]TokenSpec) TokenStep {
	for _, name := range sortedNames(current) {
		if _, ok := desired[name]; !ok {
			return TokenStep{Op: TokenOpDelete, Name: name}
		}
	}
	for _, name := range sortedNames(desired) {
		if _, ok := current[name]; !ok {
			description := desired[name].Description
			return TokenStep{Op: TokenOpCreate, Name: name, Description: &description}
		}
	}
	for _, name := range sortedNames(desired) {
		have, want := current[name], desired[name]
		if have.Status != want.Status {
			return statusStep(name, have, want)
		}
	}
	// Only descriptions differ; rule 6 handles those, so this is not reached.
	return TokenStep{}
}

// ValidateTokens checks a desired token set in the plan: it must have at least one token.
// Britive's own limits (at most 3 tokens, at most 2 ACTIVE, at least 1 ACTIVE) are not
// checked here; Britive rejects a set that breaks them with its own error during apply.
func ValidateTokens(desired map[string]TokenSpec) []string {
	if len(desired) == 0 {
		return []string{"A broker pool must have at least one token in tokens."}
	}
	return nil
}

// ApplyTokenStep returns current after step, as Britive would hold it.
func ApplyTokenStep(current map[string]TokenSpec, step TokenStep) map[string]TokenSpec {
	next := make(map[string]TokenSpec, len(current)+1)
	for k, v := range current {
		next[k] = v
	}
	switch step.Op {
	case TokenOpDelete:
		delete(next, step.Name)
	case TokenOpCreate:
		spec := TokenSpec{Status: TokenStatusInactive}
		if step.Description != nil {
			spec.Description = *step.Description
		}
		next[step.Name] = spec
	case TokenOpUpdate:
		spec := next[step.Name]
		if step.Status != nil {
			spec.Status = *step.Status
		}
		if step.Description != nil {
			spec.Description = *step.Description
		}
		next[step.Name] = spec
	}
	return next
}

// TokensMatch reports whether two token sets are equal.
func TokensMatch(a, b map[string]TokenSpec) bool {
	if len(a) != len(b) {
		return false
	}
	for name, spec := range a {
		if other, ok := b[name]; !ok || other != spec {
			return false
		}
	}
	return true
}

// TokenID builds the id of a token in the pool resource's tokens map: {pool_id}/{name}. The
// ephemeral resource takes it in token_id.
func TokenID(poolID, name string) string {
	return poolID + "/" + name
}

// tokenSetAllowed reports whether Britive allows a pool to hold tokens: at most 3 tokens, and
// 1 or 2 of them ACTIVE.
func tokenSetAllowed(tokens map[string]TokenSpec) bool {
	active := countActive(tokens)
	return len(tokens) <= MaxTokens && active >= 1 && active <= MaxActiveTokens
}

func statusStep(name string, have, want TokenSpec) TokenStep {
	status := want.Status
	step := TokenStep{Op: TokenOpUpdate, Name: name, Status: &status}
	if have.Description != want.Description {
		description := want.Description
		step.Description = &description
	}
	return step
}

func countActive(tokens map[string]TokenSpec) int {
	n := 0
	for _, t := range tokens {
		if t.Status == TokenStatusActive {
			n++
		}
	}
	return n
}

func sortedNames(tokens map[string]TokenSpec) []string {
	names := make([]string, 0, len(tokens))
	for name := range tokens {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
