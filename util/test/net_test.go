package test

import (
	"fmt"
	"testing"

	"search_engine/util"
)

func TestGetLocalIP(t *testing.T) {
	fmt.Println(util.GetLocalIP())
}

// go test -v ./util/test -run=^TestGetLocalIP$ -count=1
