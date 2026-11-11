//go:build windows

package main

import (
	"errors"
	"os"
	"strings"
	"syscall"

	"golang.org/x/sys/windows/registry"
)

const dailyFlowRunKey = `Software\Microsoft\Windows\CurrentVersion\Run`
const dailyFlowRunValue = "DailyFlow"

func getAutoStartEnabled() (bool, error) {
	key, err := registry.OpenKey(registry.CURRENT_USER, dailyFlowRunKey, registry.QUERY_VALUE)
	if errors.Is(err, syscall.ERROR_FILE_NOT_FOUND) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	defer key.Close()

	command, _, err := key.GetStringValue(dailyFlowRunValue)
	if errors.Is(err, syscall.ERROR_FILE_NOT_FOUND) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	exe, err := os.Executable()
	if err != nil {
		return false, err
	}
	return strings.EqualFold(strings.Trim(command, `"`), exe), nil
}

func setAutoStartEnabled(enabled bool) error {
	if !enabled {
		key, err := registry.OpenKey(registry.CURRENT_USER, dailyFlowRunKey, registry.SET_VALUE)
		if errors.Is(err, syscall.ERROR_FILE_NOT_FOUND) {
			return nil
		}
		if err != nil {
			return err
		}
		defer key.Close()
		if err := key.DeleteValue(dailyFlowRunValue); errors.Is(err, syscall.ERROR_FILE_NOT_FOUND) {
			return nil
		} else {
			return err
		}
	}

	exe, err := os.Executable()
	if err != nil {
		return err
	}
	key, _, err := registry.CreateKey(registry.CURRENT_USER, dailyFlowRunKey, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer key.Close()
	return key.SetStringValue(dailyFlowRunValue, `"`+exe+`"`)
}
