package away

// stranger is declared here but only another file refers to it.
type stranger interface{ Do() }

// Doer is declared where it is first used and reused elsewhere: not flagged.
type Doer interface{ Do() }

func fetch(d Doer) { d.Do() }
