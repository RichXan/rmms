package main

import (
	"fmt"
	"time"
)

func main() {
	now := time.Now().In(time.FixedZone("CST", 8*3600)).Format("2006-01-02@15:04")
	fmt.Println(now)
}
