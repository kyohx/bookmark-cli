package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

func PrintJSON(v any) error {
	return PrintJSONTo(os.Stdout, v)
}

func PrintJSONTo(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func PrintLine(format string, a ...any) {
	fmt.Fprintf(os.Stdout, format+"\n", a...)
}
