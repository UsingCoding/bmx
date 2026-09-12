package usecase

import (
	"io"

	"github.com/manifoldco/promptui"
)

type nopWriteCloser struct {
	io.Writer
}

func (nopWriteCloser) Close() error {
	return nil
}

func selectPrompt(label string, items []string, in io.Reader, out io.Writer) (int, error) {
	index, _, err := (&promptui.Select{
		Label:  label,
		Items:  items,
		Stdin:  io.NopCloser(in),
		Stdout: nopWriteCloser{Writer: out},
	}).Run()
	return index, err
}

func confirmPrompt(label string, in io.Reader, out io.Writer) (bool, error) {
	_, err := (&promptui.Prompt{
		Label:     label,
		IsConfirm: true,
		Stdin:     io.NopCloser(in),
		Stdout:    nopWriteCloser{Writer: out},
	}).Run()
	if err == promptui.ErrAbort || err == promptui.ErrEOF {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
