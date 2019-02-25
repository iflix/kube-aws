package model

import (
	"regexp"
)

type PodResources struct {
	Cpu    string `yaml:"cpu"`
	Memory string `yaml:"memory"`
}

func (r PodResources) IsValid() bool {
	match, _ := regexp.MatchString("([0-9]+m)$", r.Cpu)
	if !match {
		return false
	}

	match, _ = regexp.MatchString("([0-9]+[M|G]i)$", r.Memory)
	if !match {
		return false
	}

	return true
}
