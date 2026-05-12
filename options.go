package rest

import (
	"gorm.io/gorm"
)

type (
	options struct {
		db           *gorm.DB
		moduleName   string
		enableTenant bool
		scenarios    []string
	}

	Option func(*options)
)

func newOptions(cbs ...Option) *options {
	opts := &options{}
	for _, cb := range cbs {
		cb(opts)
	}
	return opts
}

func WithDB(db *gorm.DB) Option {
	return func(opts *options) {
		opts.db = db
	}
}

func WithModuleName(moduleName string) Option {
	return func(opts *options) {
		opts.moduleName = moduleName
	}
}

func WithTenant() Option {
	return func(opts *options) {
		opts.enableTenant = true
	}
}

func WithScenarios(scenarios ...string) Option {
	return func(opts *options) {
		opts.scenarios = scenarios
	}
}
