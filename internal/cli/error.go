package cli

import "fmt"

func WrapUserError(message string, err error) error {
	if err == nil {
		return fmt.Errorf(message)
	}
	return fmt.Errorf("%s: %w", message, err)
}
