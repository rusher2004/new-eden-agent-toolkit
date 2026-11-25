package cli

// Paths links each operation name to its command path: one or more groups,
// then the leaf. It is the only place command naming lives; Build fails for a
// described operation with no entry here, or when two operations share a path.
var Paths = map[string][]string{
	"list_character_loyalty_points": {"loyalty", "points"},
	"list_loyalty_store_offers":     {"loyalty", "store-offers"},
}

// Groups is each group's one-line help, keyed by its path joined with spaces
// ("loyalty", or "market hub" for a nested group). Build fails for a group in
// Paths with no entry here.
var Groups = map[string]string{
	"loyalty": "loyalty point balances and loyalty stores",
}
