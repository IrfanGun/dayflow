//go:build !windows

package main

import "errors"

func protectAISecret([]byte) ([]byte, error) {
	return nil, errors.New("secure AI key storage is only available on Windows")
}

func unprotectAISecret([]byte) ([]byte, error) {
	return nil, errors.New("secure AI key storage is only available on Windows")
}
