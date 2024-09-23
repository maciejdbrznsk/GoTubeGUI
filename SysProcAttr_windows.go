//go:build windows
// +build windows

package main

import "syscall"

func getOSSysProcAttr() *syscall.SysProcAttr {
	// Hide CMD on Windows
	return &syscall.SysProcAttr{HideWindow: true}
}
