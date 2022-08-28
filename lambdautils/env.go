package lambdautils

import (
	"fmt"
	"os"
)

// DDB_TABLE is the environment variable that contains the Dynamodb table name.
// TODO: Can we use Golang build constraints to change this value?
const EnvDDBtable = "DDB_TABLE"

// Mustenv ensures an environment variable is set and panics if it is not.
func Mustenv(names ...string) {
	for _, name := range names {
		if os.Getenv(name) == "" {
			panic(fmt.Errorf("missing required environment variable: %v", name))
		}
	}
}

// DDBtable returns the Dynamodb table name set in the environment variable.
func DDBtable() string {
	return os.Getenv(EnvDDBtable)
}
