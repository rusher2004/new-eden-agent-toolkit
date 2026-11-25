package sde

// Client is a concrete producer type.
type Client struct{}

// Lister is a producer-side interface meant for callers.
type Lister interface {
	List() error
}
