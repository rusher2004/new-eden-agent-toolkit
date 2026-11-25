package dup

// Runner is declared in two directories sharing a package name.
type Runner interface{ Run() }

// A gives the file code, so only the interface test sees it as a type.
func A() {}
