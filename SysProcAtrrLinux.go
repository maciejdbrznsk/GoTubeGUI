//go:build linux
// +build linux

package main

import "syscall"

func getOSSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{}
}
