// Package commands owns user-facing command policy and orchestration.
package commands

import (
	"context"
	"io"
	"vbx/internal/backend/lima"
	"vbx/internal/command"
	"vbx/internal/state"
	"vbx/internal/workspace"
)

type readiness interface {
	EnsureReady(context.Context, string) (workspace.Ready, error)
}
type scopeAcquirer interface {
	Acquire(context.Context, string) (*workspace.Scope, error)
}
type instanceLister interface {
	List(context.Context) ([]lima.Instance, error)
}
type stopper interface {
	Stop(context.Context, string) error
}
type lifecycleBackend interface {
	stopper
	Status(context.Context, string) (string, error)
	Delete(context.Context, string) error
}
type processRunner interface {
	Run(context.Context, io.Reader, io.Writer, io.Writer, string, ...string) error
}
type replacer interface{ Replace(string, []string) error }
type lineReader interface {
	ReadLine(context.Context, string) (string, error)
}
type hostChecks interface{ Check(context.Context) error }

// Dependencies supplies construction inputs; handlers retain only what they use.
type Dependencies struct {
	Ready       readiness
	Scopes      scopeAcquirer
	Inventory   instanceLister
	Lima        lifecycleBackend
	Replacer    replacer
	Runner      processRunner
	Host        hostChecks
	Interaction lineReader
	Paths       state.Paths
	Out, Err    io.Writer
}
type Factory func(Dependencies) command.Command

var factories []Factory

func register(f Factory) { factories = append(factories, f) }
func NewWith(d Dependencies, fs []Factory) []command.Command {
	out := make([]command.Command, 0, len(fs))
	for _, f := range fs {
		out = append(out, f(d))
	}
	return out
}
func New(d Dependencies) []command.Command { return NewWith(d, factories) }
