//go:build windows

package main

import "syscall"

func syscallSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
	}
}

func unixSysProcAttr() *syscall.SysProcAttr {
	return nil
}
