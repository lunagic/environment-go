package environment

import (
	"errors"
	"io"
	"os"
	"reflect"
	"strconv"
	"strings"
)

func New() *Service {
	env := NewEmpty()

	// Load in the existing system variables
	for _, e := range os.Environ() {
		pair := strings.SplitN(e, "=", 2)
		env.vars[pair[0]] = pair[1]
	}

	// Load in local files
	for _, fileToCheck := range []string{
		".env.local",
		".env",
	} {
		envFile, err := os.Open(fileToCheck)
		if err == nil {
			if err := env.Parse(envFile); err != nil {
				panic(err)
			}
		}
	}

	return env
}

func NewEmpty() *Service {
	return &Service{
		vars: map[string]string{},
	}
}

type Service struct {
	vars map[string]string
}

func (env Service) Decode(target any) error {
	targetInstance := reflect.ValueOf(target)
	if targetInstance.Kind() != reflect.Pointer {
		return errors.New("must provider pointer")
	}

	targetInstance = targetInstance.Elem()

	for i := 0; i < targetInstance.NumField(); i++ {
		fieldInstance := targetInstance.Field(i)
		fieldDefinition := targetInstance.Type().Field(i)

		tag := fieldDefinition.Tag.Get("env")
		if tag == "" {
			continue
		}

		valueFromEnv, ok := env.vars[tag]
		if !ok {
			continue
		}

		switch fieldDefinition.Type.Kind() {
		case reflect.String:
			fieldInstance.SetString(valueFromEnv)
		case reflect.Bool:
			convertedValue, err := strconv.ParseBool(valueFromEnv)
			if err != nil {
				return err
			}

			fieldInstance.SetBool(convertedValue)
		case reflect.Int:
			convertedValue, err := strconv.Atoi(valueFromEnv)
			if err != nil {
				return err
			}

			fieldInstance.SetInt(int64(convertedValue))
		}
	}

	return nil
}

func (s Service) Parse(reader io.Reader) error {
	bodyBytes, err := io.ReadAll(reader)
	if err != nil {
		return err
	}

	for _, line := range strings.Split(string(bodyBytes), "\n") {
		parts := strings.SplitN(line, "=", 2)
		if len(parts) < 2 {
			continue
		}

		key := parts[0]
		value := parts[1]

		// Skip the env variable is already set
		if _, alreadySet := s.vars[key]; alreadySet {
			continue
		}

		// line = strings.Trim(line, "\"'")

		// Set the value
		s.vars[key] = value
	}

	return nil
}
