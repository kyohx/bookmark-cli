package cli

import (
	"encoding/json"
	"fmt"
	"os"
)

func PrintJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func PrintLine(format string, a ...any) {
	fmt.Fprintf(os.Stdout, format+"\n", a...)
}
