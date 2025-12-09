package testdata

import (
	"context"
	"errors"
	"fmt"
)

// FunctionWithNamedReturns demonstrates named return values.
func FunctionWithNamedReturns(ctx context.Context, input string) (result string, err error) {
	if input == "" {
		err = errors.New("input is empty")
		return result, err
	}

	result = fmt.Sprintf("processed: %s", input)
	return result, err
}

// FunctionWithUnnamedReturns has no named return values.
func FunctionWithUnnamedReturns(input string) (string, error) {
	if input == "" {
		return "", errors.New("input is empty")
	}
	return fmt.Sprintf("processed: %s", input), nil
}

// FunctionWithErrorHandling demonstrates error checking.
func FunctionWithErrorHandling(value string) (result string, err error) {
	processed, err := process(value)
	if err != nil {
		err = fmt.Errorf("processing failed: %w", err)
		return result, err
	}

	result = processed
	return result, err
}

func process(input string) (output string, err error) {
	if input == "" {
		err = errors.New("empty input")
		return output, err
	}
	output = input
	return output, err
}

// FunctionNoErrorHandling does not check errors.
func FunctionNoErrorHandling(value string) (result string) {
	result = value
	return result
}

// FunctionNoReturns has no return values.
func FunctionNoReturns() {
	// Do nothing
}

// ComplexFunction has multiple parameters and return values.
func ComplexFunction(ctx context.Context, id int, name string, active bool) (result string, count int, err error) {
	if id < 0 {
		err = errors.New("invalid id")
		return result, count, err
	}

	if name == "" {
		err = errors.New("name required")
		return result, count, err
	}

	result = fmt.Sprintf("ID:%d Name:%s Active:%v", id, name, active)
	count = len(result)

	return result, count, err
}
