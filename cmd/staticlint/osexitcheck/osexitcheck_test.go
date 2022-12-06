package osexitcheck

import (
	"golang.org/x/tools/go/analysis/analysistest"
	"testing"
)

func Test(t *testing.T) {
	testdata := analysistest.TestData()

	analysistest.Run(t, testdata, Analyzer, "a", "b", "c")
}
