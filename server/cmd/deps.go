// Package main 依赖声明已迁移至 server/tools.go（//go:build tools 模式）。
//
// tools.go 使用构建约束排除在生产二进制之外，仅用于 go mod tidy 依赖跟踪。
// 此文件保留作为包声明，避免 main 包为空导致编译错误。
package main

