package utils

import (
	"fmt"
	"hash/fnv"
)

func HashNode(s string, lenNodes int) uint32 {
	h := fnv.New32a()
	h.Write([]byte(s))
	nodeId := h.Sum32() % uint32(lenNodes)
	fmt.Println("Node ID: ", nodeId)
	return h.Sum32() % uint32(lenNodes)
}
