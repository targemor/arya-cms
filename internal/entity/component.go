package entity

import "fmt"

// Component is a reusable schema of attributes embedded in ContentTypes.
type Component struct {
	Name        string      `json:"name"                  example:"contact_data"`
	Category    string      `json:"category"              example:"shared"`
	DisplayName string      `json:"displayName,omitempty" example:"Contact Data"`
	Icon        string      `json:"icon,omitempty"        example:"phone"`
	Attributes  []Attribute `json:"attributes"`
} // @name entity.Component

// UID returns the unique identifier of the component: <category>.<name> or <name>.
func (c *Component) UID() string {
	if c.Category != "" {
		return c.Category + "." + c.Name
	}

	return c.Name
}

// GetDisplayName returns c.DisplayName, defaulting to c.Name when empty.
func (c *Component) GetDisplayName() string {
	if c.DisplayName == "" {
		return c.Name
	}

	return c.DisplayName
}

// Validate checks the component definition is valid. Every error wraps ErrInvalidComponent.
func (c *Component) Validate() error {
	if !isIdentifier(c.Name) {
		return fmt.Errorf("%w: component name %q", ErrInvalidComponent, c.Name)
	}

	if c.Category != "" && !isIdentifier(c.Category) {
		return fmt.Errorf("%w: component category %q", ErrInvalidComponent, c.Category)
	}

	if len(c.Attributes) == 0 || len(c.Attributes) > _maxAttributes {
		return fmt.Errorf("%w: %d attributes", ErrInvalidComponent, len(c.Attributes))
	}

	seen := make(map[string]struct{}, len(c.Attributes))

	for i := range c.Attributes {
		if err := c.Attributes[i].validate(); err != nil {
			return fmt.Errorf("%w: %w", ErrInvalidComponent, err)
		}

		if _, dup := seen[c.Attributes[i].Name]; dup {
			return fmt.Errorf("%w: duplicate attribute %q", ErrInvalidComponent, c.Attributes[i].Name)
		}

		seen[c.Attributes[i].Name] = struct{}{}
	}

	return nil
}
