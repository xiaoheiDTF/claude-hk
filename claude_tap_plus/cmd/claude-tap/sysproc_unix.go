//go:build !windows

package main

import "syscall"

func syscallSysProcAttr() *syscall.SysProcAttr {
	return nil
}

func unixSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		Setpgid: true,
	}
}
