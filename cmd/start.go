package cmd

import (
	"fmt"
	"sync"
)

type OneAndOnlyNumber struct {
	num        int
	generation int
	numMutex   sync.RWMutex
}

const MembersToNotify = 2

func InitTheNumber(val int) *OneAndOnlyNumber {
	return &OneAndOnlyNumber{
		num: val,
	}
}

func (n *OneAndOnlyNumber) SetValue(newVal int) {
	fmt.Println("SetValue: ", newVal)
	n.numMutex.Lock()
	defer n.numMutex.Unlock()
	n.num = newVal
	n.generation = n.generation + 1
}

func (n *OneAndOnlyNumber) GetValue() (int, int) {
	n.numMutex.RLock()
	defer n.numMutex.RUnlock()
	return n.num, n.generation
}

func (n *OneAndOnlyNumber) NotifyValue(curVal int, curGeneration int) bool {
	if curGeneration > n.generation {
		n.numMutex.Lock()
		defer n.numMutex.Unlock()
		n.generation = curGeneration
		n.num = curVal
		return true
	}
	return false
}
