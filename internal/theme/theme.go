// Package theme defines cacheriff's semantic color palette. Every
// color is a plain hex string with no dependency on the rendering
// library, so the same definitions can be reused as-is by a future
// Neovim port, where hex values map directly onto `guifg`/`guibg` in
// highlight groups.
package theme

// Theme is a named set of semantic color roles.
type Theme struct {
	Name string

	// Primary accents titles, section headers, and the sidebar's
	// selected entry. Neovim equivalent: link to Title.
	Primary string

	// ActiveBorder marks the currently focused panel's border.
	// Neovim equivalent: a custom "CacheriffBorderActive" group.
	ActiveBorder string

	// InactiveBorder marks an unfocused panel's border.
	// Neovim equivalent: link to FloatBorder.
	InactiveBorder string

	// Muted is used for disabled/unavailable items, e.g. a package
	// manager that isn't installed. Neovim equivalent: link to Comment.
	Muted string

	// Faint is the least prominent text: the help bar, metadata.
	// Neovim equivalent: link to NonText.
	Faint string

	// Error marks failures (a load error, a failed removal).
	// Neovim equivalent: link to DiagnosticError.
	Error string

	// Success marks completed/positive state.
	// Neovim equivalent: link to DiagnosticOk.
	Success string
}

// Default is cacheriff's built-in color scheme, used for any role the
// user hasn't overridden.
var Default = Theme{
	Name:           "cacheriff-dark",
	Primary:        "#FF5FAF",
	ActiveBorder:   "#FF87D7",
	InactiveBorder: "#585858",
	Muted:          "#585858",
	Faint:          "#808080",
	Error:          "#FF5F5F",
	Success:        "#87D787",
}

// Override holds user-supplied color overrides; any empty field falls
// back to the base theme. It mirrors Theme field-for-field so a new
// role only needs to be added in one place.
type Override struct {
	Primary        string `yaml:"primaryColor,omitempty"`
	ActiveBorder   string `yaml:"activeBorderColor,omitempty"`
	InactiveBorder string `yaml:"inactiveBorderColor,omitempty"`
	Muted          string `yaml:"mutedColor,omitempty"`
	Faint          string `yaml:"faintColor,omitempty"`
	Error          string `yaml:"errorColor,omitempty"`
	Success        string `yaml:"successColor,omitempty"`
}

// Apply returns base with every non-empty field in o overlaid on top.
func (o Override) Apply(base Theme) Theme {
	if o.Primary != "" {
		base.Primary = o.Primary
	}
	if o.ActiveBorder != "" {
		base.ActiveBorder = o.ActiveBorder
	}
	if o.InactiveBorder != "" {
		base.InactiveBorder = o.InactiveBorder
	}
	if o.Muted != "" {
		base.Muted = o.Muted
	}
	if o.Faint != "" {
		base.Faint = o.Faint
	}
	if o.Error != "" {
		base.Error = o.Error
	}
	if o.Success != "" {
		base.Success = o.Success
	}
	return base
}
