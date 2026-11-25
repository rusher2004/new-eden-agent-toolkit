package commands

import (
	"example.com/m/sde"
	hz "example.com/m/sde"
)

type holder struct {
	H *sde.Client
}

var hook func(c *hz.Client)

// Trigger takes the concrete client.
func Trigger(c *sde.Client, n int) error { return nil }

// local is a consumer-declared interface; it is fine.
type local interface{ Do() }

var _ = sde.Lister(nil)

var factory = func() (*sde.Client, error) { return sde.New("x"), nil }
