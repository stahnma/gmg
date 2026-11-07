package config

import (
	"errors"
	"fmt"
	"io/fs"

	"github.com/joho/godotenv"
)

// DefaultEnvFile is read from the working directory when no file is named.
const DefaultEnvFile = ".env"

// WithEnvFile returns a getenv for Load that looks up each variable in the
// process environment first (via lookup, usually os.LookupEnv) and falls back
// to the values in the dotenv file at path. A variable that is set in the
// environment, even to an empty string, wins over the file.
//
// If path does not exist, the result is the environment alone, unless required
// is true, in which case it is an error. loaded reports whether the file was read.
func WithEnvFile(lookup func(string) (string, bool), path string, required bool) (getenv func(string) string, loaded bool, err error) {
	fromEnv := func(k string) string { v, _ := lookup(k); return v }
	file, err := godotenv.Read(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) && !required {
			return fromEnv, false, nil
		}
		return nil, false, fmt.Errorf("reading %s: %w", path, err)
	}
	return func(k string) string {
		if v, ok := lookup(k); ok {
			return v
		}
		return file[k]
	}, true, nil
}
