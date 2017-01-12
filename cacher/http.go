package cacher

import (
	"bufio"
	"fmt"
	"io"
)

// WriteHTTP writes cache data in http format
func WriteHTTP(w io.Writer, input *Input) error {
	bw := bufio.NewWriter(w)
	defer bw.Flush()

	bw.WriteString(fmt.Sprintf("HTTP %d\n", input.StatusCode))

	if input.URL != nil {
		bw.WriteString(fmt.Sprintf("X-Mirrored-Url: %s\n", input.URL.String()))
	}

	if input.StatusCode >= 200 && input.StatusCode <= 299 {
		writeHTTP2xx(bw, input)
	} else if input.StatusCode >= 300 && input.StatusCode <= 399 {
		writeHTTP3xx(bw, input)
	}

	return nil
}

func writeHTTP2xx(bw *bufio.Writer, input *Input) {
	if len(input.ContentType) > 0 {
		bw.WriteString(fmt.Sprintf("Content-Type: %s\n", input.ContentType))
	}

	bodyLen := len(input.Body)
	if bodyLen > 0 {
		bw.WriteString(fmt.Sprintf("Content-Length: %d\n", bodyLen))
		bw.WriteString("\n")
		bw.WriteString(input.Body)
	}
}

func writeHTTP3xx(bw *bufio.Writer, input *Input) {
	if input.Redirection != nil {
		bw.WriteString(fmt.Sprintf("Location: %s\n", input.Redirection.String()))
		return
	}
}
