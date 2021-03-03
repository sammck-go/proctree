package proctree

// Config provides configuration options for contruction of a ProcTree.  The constructed object is immutable
// after it is constructed by NewConfig.
type Config struct {
	// includeKernelThreads enables inclusion of kernel threads (children of pid 2). By default, kernel threads are
	// excluded.
	includeKernelThreads bool

	// includeRootAncestors enables inclusion of all Processes that are ancestors of the pids configured in rootPids.
	// Has no effect if rootPids are not specified.
	includeRootAncestors bool

	// rootPids list a list of pids to use as roots of the process tree. If omitted, all orphaned processes are
	// used as roots.
	rootPids []int
}

// ConfigOption is an opaque configuration option setter created by one of the With functions.
// It follows the Golang "options" pattern.
type ConfigOption func(*Config)

const (
	defaultIncludeKernelThreads = false
	defaultIncludeRootAncestors = false
)

// NewConfig creates a proctree Config object from provided options. The resulting object
// can be passed to New using WithConfig.
func NewConfig(opts ...ConfigOption) *Config {
	cfg := &Config{
		includeKernelThreads: defaultIncludeKernelThreads,
		includeRootAncestors: defaultIncludeRootAncestors,
		rootPids:             []int{},
	}

	for _, opt := range opts {
		opt(cfg)
	}

	return cfg
}

// WithConfig allows initialization of a new configuration object starting with an existing one,
// and incremental initialization of configuration separately from initialization of the PidFile.
// If provided, this option should be appear first in the option list, since it replaces all
// configuration values.
func WithConfig(other *Config) ConfigOption {
	return func(cfg *Config) {
		cfg.includeKernelThreads = other.includeKernelThreads
		cfg.includeRootAncestors = other.includeRootAncestors
		cfg.rootPids = make([]int, len(other.rootPids))
		copy(cfg.rootPids, other.rootPids)
	}
}

// Refine creates a new Config object by applying ConfigOptions to an existing config.
func (cfg *Config) Refine(opts ...ConfigOption) *Config {
	newOpts := append([]ConfigOption{WithConfig(cfg)}, opts...)
	newConfig := NewConfig(newOpts...)
	return newConfig
}

// WithKernelThreads enables inclusion of kernel threads (children of pid 2). By default, kernel threads are
// excluded.
func WithKernelThreads() ConfigOption {
	return func(cfg *Config) {
		cfg.includeKernelThreads = true
	}
}

// WithoutKernelThreads disables inclusion of kernel threads (children of pid 2). This is the default option.
func WithoutKernelThreads() ConfigOption {
	return func(cfg *Config) {
		cfg.includeKernelThreads = false
	}
}

// WithRootAncestors enables inclusion of all Processes that are ancestors of the pids configured with WithRootPid.
// Has no effect if WithRootPid is not specified. By default, root ancestors are excluded and the root Processes
// appear at the first level of the tree.
func WithRootAncestors() ConfigOption {
	return func(cfg *Config) {
		cfg.includeRootAncestors = true
	}
}

// WithoutRootAncestors disables inclusion of Processes that are ancestors of the pids configured with WithRootPid.
// Has no effect if WithRootPid is not specified. Root ancestors are excluded and the root Processes
// appear at the first level of the tree. This is the default setting.
func WithoutRootAncestors() ConfigOption {
	return func(cfg *Config) {
		cfg.includeRootAncestors = false
	}
}

// WithRootPid adds a pid to the set of pids to be included as roots of the tree. By default, all orphaned processes are
// included as roots.
func WithRootPid(pid int) ConfigOption {
	return func(cfg *Config) {
		cfg.rootPids = append(cfg.rootPids, pid)
	}
}

// WithoutRootPid removes all pids added with WithRootPid, restoring config the default, which is to include all orphaned processses.
func WithoutRootPid() ConfigOption {
	return func(cfg *Config) {
		cfg.rootPids = []int{}
	}
}
