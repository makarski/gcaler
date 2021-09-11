package userio

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

const (
	yesInput = "y"
)

var (
	out = os.Stdout
	in  = os.Stdin

	unexpectedNewlineErr = errors.New("unexpected newline")
)

func UserIn(buf io.Reader) (string, error) {
	if _, err := io.Copy(out, buf); err != nil {
		return "", err
	}

	scanner := bufio.NewScanner(in)
	if ok := scanner.Scan(); !ok {
		return "", scanner.Err()
	}
	input := scanner.Text()

	return input, nil
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
	if err != nil {
		if err == unexpectedNewlineErr {
			return false, nil
		}
		return false, err
	}

	if strings.ToLower(input) == yesInput {
		return true, nil
	}

	return false, nil
}
