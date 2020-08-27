package main

import (
	"encoding/hex"
	"testing"
)

func TestABI(t *testing.T) {
	data,_ := hex.DecodeString("a9059cbb000000000000000000000000ea0e2dc7d65a50e77fc7e84bff3fd2a9e781ff5c0000000000000000000000000000000000000000000000015af1d78b58c40000")
	parse(data)
}