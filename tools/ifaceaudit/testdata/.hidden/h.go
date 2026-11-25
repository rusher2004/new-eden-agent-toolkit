package hidden

import "example.com/m/sde"

// Copy would be reported if hidden directories were scanned.
func Copy(c *sde.Client) {}
