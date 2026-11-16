package interaction

import (
	"context"
	"fmt"
	"io"
	"strings"
)

type Terminal struct {
	In    io.Reader
	Out   io.Writer
	IsTTY func() bool
}

func (t Terminal) ReadLine(ctx context.Context, prompt string) (string, error) {
	if !t.IsTTY() {
		return "", fmt.Errorf("interactive approval requires a terminal")
	}
	if ctx.Err() != nil {
		return "", context.Cause(ctx)
	}
	fmt.Fprint(t.Out, prompt)
	type response struct {
		text string
		err  error
	}
	result := make(chan response, 1)
	go func() {
		var b strings.Builder
		one := make([]byte, 1)
		for {
			n, err := t.In.Read(one)
			if n > 0 {
				if one[0] == '\n' {
					result <- response{text: b.String()}
					return
				}
				b.WriteByte(one[0])
			}
			if err != nil {
				result <- response{err: err}
				return
			}
		}
	}()
	var answer string
	select {
	case r := <-result:
		if r.err != nil {
			return "", r.err
		}
		answer = strings.TrimSpace(r.text)
	case <-ctx.Done():
		return "", context.Cause(ctx)
	}

	return answer, nil
}
