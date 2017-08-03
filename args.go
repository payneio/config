package config

import (
	"fmt"
	"os"
	"strings"
)

// os.Args splits by space
// let's parse into pairs
type argPair struct {
	Key string
	Val string
}

func loadCommandLineArgs() {
	pairs := parseCommandLineArgs()
	for _, p := range pairs {
		key, _ := stripConfigPrefix(p.Key)
		Set(key, p.Val)
	}
}

// parseCommandLineArgs parses all the os.Args into key-value pairs
// according to the http://docopt.org standard. E.g., each of these
// will work:
//   -k (short flag, set to true)
//   -k value (short flag set to value)
//   -klm  (Stacked short flags. Each set to true)
//   -klm value (k and l set to true, m set to value)
//   --key (flag `key` set to true)
//   --key value
//   --key=value
// All keys will be lower-cased
func parseCommandLineArgs() []argPair {

	// We use a referenceable list of pairs during
	// our pair construction. At the end, we'll
	// convert to values
	var pairs []*argPair
	lastKeyUsedZeroValue := false

	// Every arg in the command line is positional until we reach an option or
	// flag
	doneWithPositionalArgs := false

	// Run through all the args (minus the program name)
	for _, arg := range os.Args[1:] {

		// The general strategy is to create a pair from
		// an arg if we can (e.g. contains an equal rune)
		// but if the arg is indefinite, we just set the
		// value of the pair to empty string.

		// If we have an arg without a hypen, we'll do
		// a look-back and use the arg to set the value
		// of the last pair (if it has an empty string val)

		// if --
		if strings.HasPrefix(arg, "--") {
			doneWithPositionalArgs = true
			rawArg := strings.TrimPrefix(arg, "--")
			// if include =, split into key/val
			parts := strings.SplitN(rawArg, "=", 2)
			if len(parts) == 1 {
				lastKeyUsedZeroValue = true
				newPair := &argPair{rawArg, ""}
				pairs = append(pairs, newPair)
			} else {
				lastKeyUsedZeroValue = false
				newPair := &argPair{parts[0], parts[1]}
				pairs = append(pairs, newPair)
			}

		} else if strings.HasPrefix(arg, "-") {
			doneWithPositionalArgs = true
			// Short flags
			// Single hyphens behave a bit differently as they
			// can "stack" as boolean values. If you put several
			// together, like -abc then all then a and b should
			// be set to true, and c should be left indefinite
			rawArg := strings.TrimPrefix(arg, "-")

			// strip off anything after an equal rune, to set
			// on the last short flag
			parts := strings.SplitN(rawArg, "=", 2)
			rawArg = parts[0]

			// make pair for each short flag and
			for _, c := range rawArg {

				// if we get another hypen, just ignore it
				if c == rune('-') {
					continue
				}

				// Set all short flags to "1"/true.
				// If there was an argument, we'll overwrite
				// the value for the last flag
				val := "1"
				lastKeyUsedZeroValue = true
				newPair := &argPair{fmt.Sprintf("%c", c), val}
				pairs = append(pairs, newPair)
			}

			// Now handle the equal rune (a value set on the short flag)
			// Set the last pair to the value
			if len(parts) > 1 {
				pairs[len(pairs)-1].Val = parts[1]
			}

		} else {

			// for the purposes of config, just skip over positional args
			if !doneWithPositionalArgs {
				continue
			}

			// This is a value, not a flag (since it doesn't start with a hyphen)
			// Set as val of prev pair, if the pair value is empty
			last := pairs[len(pairs)-1]
			if lastKeyUsedZeroValue {
				last.Val = arg
			}
		}
	}

	// Package up for return value
	var returnPairs []argPair
	for _, p := range pairs {
		p.Key = strings.ToLower(p.Key)
		// Any remaining unassigned values should be set to "1" (true)
		if p.Val == "" {
			p.Val = "1"
		}
		returnPairs = append(returnPairs, *p)
	}
	return returnPairs
}
