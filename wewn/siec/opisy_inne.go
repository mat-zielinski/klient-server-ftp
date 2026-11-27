//go:build !windows

package siec

// opisyInterfejsow: poza Windows brak opisów, rozpoznanie WireGuard tylko po nazwie.
func opisyInterfejsow() map[int]string { return map[int]string{} }
