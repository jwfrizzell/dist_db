package cmd

import (
	"sync"
)

type OneAndOnlyNumber struct {
	num        int
	generation int
	numMutext  sync.RWMutex
}

const MembersToNotify = 2

func InitTheNumber(val int) *OneAndOnlyNumber {
	return &OneAndOnlyNumber{
		num: val,
	}
}

func (n *OneAndOnlyNumber) SetValue(newVal int) {
	n.numMutext.Lock()
	defer n.numMutext.Unlock()
	n.num = newVal
	n.generation = n.generation + 1
}

func (n *OneAndOnlyNumber) GetValue() (int, int) {
	n.numMutext.RLock()
	defer n.numMutext.Unlock()
	return n.num, n.generation
}

func (n *OneAndOnlyNumber) NotifyValue(curVal int, curGeneration int) bool {
	if curGeneration > n.generation {
		n.numMutext.Lock()
		defer n.numMutext.Unlock()
		n.generation = curGeneration
		n.num = curVal
		return true
	}
	return false
}
