package sde

import (
	"strconv"
	"strings"
)

// gameName writes a celestial's or station's name the way the game shows
// it. fuzzwork's dump numbers planets in Arabic ("Jita 4 - Moon 4 - Caldari
// Navy Assembly Plant"), while the game and ESI use Roman numerals for the
// planet and Arabic for everything after it ("Jita IV - Moon 4 - ...").
// Only the number right after the system name is a planet's, so the
// system's own name, which can end in digits, is never touched. A name
// already in the game's form, or one not starting with the system's name,
// comes back unchanged.
func gameName(name, system string) string {
	if system == "" || !strings.HasPrefix(name, system+" ") {
		return name
	}
	rest := name[len(system)+1:]
	num, tail, _ := strings.Cut(rest, " ")
	n, err := strconv.Atoi(num)
	if err != nil || n <= 0 || strconv.Itoa(n) != num {
		return name
	}
	if tail != "" {
		tail = " " + tail
	}
	return system + " " + roman(n) + tail
}

// roman writes n (1 to 3999) in Roman numerals.
func roman(n int) string {
	vals := []int{1000, 900, 500, 400, 100, 90, 50, 40, 10, 9, 5, 4, 1}
	syms := []string{"M", "CM", "D", "CD", "C", "XC", "L", "XL", "X", "IX", "V", "IV", "I"}
	var b strings.Builder
	for i, v := range vals {
		for n >= v {
			b.WriteString(syms[i])
			n -= v
		}
	}
	return b.String()
}

// systemNameSQL selects the name of the solar system whose ID is in col,
// for gameName.
func systemNameSQL(col string) string {
	return `COALESCE((SELECT solarSystemName FROM mapSolarSystems WHERE solarSystemID = ` + col + `), '')`
}
