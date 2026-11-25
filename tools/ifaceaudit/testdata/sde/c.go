package sde

// New builds the concrete client.
func New(name string) *Client { return &Client{} }

// Record is a plain data struct: no methods, no constructor.
type Record struct{ ID int }

// internalSeam is unexported: internal to the producer, not meant for callers.
type internalSeam interface{ Seam() }
