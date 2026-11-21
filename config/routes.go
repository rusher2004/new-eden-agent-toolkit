package config

// Route is a saved jump route, by name: the systems in order, from origin to
// destination, with the user's notes on why it goes that way.
type Route struct {
	Name string `json:"name" yaml:"name"`
	// ShipClass is the kind of ship the route was planned for, such as
	// jump_freighter. Free text for now.
	ShipClass string          `json:"shipClass,omitempty" yaml:"ship_class,omitempty"`
	Waypoints []RouteWaypoint `json:"waypoints" yaml:"waypoints"`
	Notes     string          `json:"notes,omitempty" yaml:"notes,omitempty"`
}

// RouteWaypoint is one system on a route. The name is for reading the file;
// the ID is what neat uses.
type RouteWaypoint struct {
	ID   int64  `json:"id" yaml:"id"`
	Name string `json:"name,omitempty" yaml:"name,omitempty"`
}

// SystemIDs lists the route's systems in order.
func (r Route) SystemIDs() []int64 {
	ids := make([]int64, len(r.Waypoints))
	for i, w := range r.Waypoints {
		ids[i] = w.ID
	}
	return ids
}

func routeName(r Route) string { return r.Name }

// UpsertRoute adds or replaces (by case-insensitive name) a route.
func (c *Config) UpsertRoute(r Route) { c.Routes = upsert(c.Routes, r, routeName) }

// DeleteRoute removes a route by name.
func (c *Config) DeleteRoute(name string) error {
	var err error
	c.Routes, err = remove(c.Routes, name, routeName)
	return err
}

// Route looks up a route by name.
func (c Config) Route(name string) (Route, bool) { return find(c.Routes, name, routeName) }
