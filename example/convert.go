package main

import (
	"github.com/shihuafan/convert/example/a"
	"github.com/shihuafan/convert/example/b"
)

//go:generate convert -from=a.A -to=b.B
func ConvertAToB(a *a.A) *b.B {
	b := &b.B{}
	if a.Age != nil {
		b.Age = int(*a.Age)
	}
	if a.High != nil {
		b.High = *a.High
	}
	b.Name = &a.Name
	return b
}
