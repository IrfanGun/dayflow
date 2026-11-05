//go:build windows

package main

import (
	"errors"
	"unsafe"

	"golang.org/x/sys/windows"
)

func protectAISecret(plain []byte) ([]byte, error) {
	if len(plain) == 0 {
		return nil, errors.New("empty secret")
	}
	in := windows.DataBlob{Size: uint32(len(plain)), Data: &plain[0]}
	var out windows.DataBlob
	if err := windows.CryptProtectData(&in, nil, nil, 0, nil, windows.CRYPTPROTECT_UI_FORBIDDEN, &out); err != nil {
		return nil, err
	}
	defer windows.LocalFree(windows.Handle(unsafe.Pointer(out.Data)))
	return append([]byte(nil), unsafe.Slice(out.Data, out.Size)...), nil
}

func unprotectAISecret(encrypted []byte) ([]byte, error) {
	if len(encrypted) == 0 {
		return nil, errors.New("empty encrypted secret")
	}
	in := windows.DataBlob{Size: uint32(len(encrypted)), Data: &encrypted[0]}
	var out windows.DataBlob
	var description *uint16
	if err := windows.CryptUnprotectData(&in, &description, nil, 0, nil, windows.CRYPTPROTECT_UI_FORBIDDEN, &out); err != nil {
		return nil, err
	}
	defer windows.LocalFree(windows.Handle(unsafe.Pointer(out.Data)))
	if description != nil {
		defer windows.LocalFree(windows.Handle(unsafe.Pointer(description)))
	}
	return append([]byte(nil), unsafe.Slice(out.Data, out.Size)...), nil
}
