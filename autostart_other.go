//go:build !windows

package main

import "errors"

func getAutoStartEnabled() (bool, error) {
	return false, errors.New("automatic startup is only supported on Windows")
}

func setAutoStartEnabled(bool) error {
	return errors.New("automatic startup is only supported on Windows")
}
