// Package command defines the boundary between CLI parsing and command execution.
package command

import "context"

type Command interface {
	Spec() Spec
	Run(context.Context, Invocation) error
}

type FlagKind uint8

const Boolean FlagKind = 0

type Flag struct {
	Summary string
	Kind    FlagKind
}
type OperandPolicy uint8

const (
	NoOperands OperandPolicy = iota
	OptionalOperands
	RequiredOperands
)

type Spec struct {
	Name, Summary, Usage string
	Flags                map[string]Flag
	Operands             OperandPolicy
	OperandHint          string
}
type GeneralOptions struct{ Project string }
type FlagValues struct{ values map[string]bool }

func NewFlagValues(values map[string]bool) FlagValues {
	copy := make(map[string]bool, len(values))
	for k, v := range values {
		copy[k] = v
	}
	return FlagValues{copy}
}
func (f FlagValues) Bool(name string) bool { return f.values[name] }
func (f FlagValues) Len() int              { return len(f.values) }

type Invocation struct {
	General GeneralOptions
	Flags   FlagValues
	Args    []string
}

func NewInvocation(g GeneralOptions, f map[string]bool, args []string) Invocation {
	return Invocation{General: g, Flags: NewFlagValues(f), Args: append([]string(nil), args...)}
}

type UsageError struct{ Message string }

func (e *UsageError) Error() string { return e.Message }

// Handler adapts a command-local execution function to the command contract.
type Handler struct {
	Definition Spec
	Execute    func(context.Context, Invocation) error
}

func (h Handler) Spec() Spec                                   { return h.Definition }
func (h Handler) Run(ctx context.Context, in Invocation) error { return h.Execute(ctx, in) }
