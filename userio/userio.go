package userio

import (
	"fmt"
	"io"
	"os"
	"strings"
)

var (
	out = os.Stdout
	in  = os.Stdin
)

func UserIn(buf io.Reader) (string, error) {
	if _, err := io.Copy(out, buf); err != nil {
		return "", err
	}

	var input string
	_, err := fmt.Fscanln(in, &input)
	return input, err
}

func UserInInt(buf io.Reader) (int, error) {
	if _, err := io.Copy(out, buf); err != nil {
		return 0, err
	}

	var input int
	_, err := fmt.Fscanf(in, "%d\n", &input)
	return input, err
}

func UserInBool(buf io.ReadWriter) (bool, error) {
	if _, err := buf.Write([]byte(" [y/N]: ")); err != nil {
		return false, err
	}

	if _, err := io.Copy(out, buf); err != nil {
		return false, err
	}

	var input string
	_, err := fmt.Fscanln(in, &input)
	if err.Error() == "unexpected newline" {
		return false, nil
	} else if err != nil {
		return false, err
	}

	if strings.ToLower(input) == "y" {
		return true, nil
	}

	return false, nil
}
