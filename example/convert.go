package main

import (
	"github.com/shihuafan/convert/example/a"
	"github.com/shihuafan/convert/example/b"
)

//go:generate convert -from=a.A -to=b.B
func ConvertAAToBB(from *a.A) *b.B {
	to := &b.B{}
	if from.Age != nil {
		to.Age = int(*from.Age)
	}
	if from.High != nil {
		to.High = *from.High
	}
	to.Name = &from.Name
	return to
}
