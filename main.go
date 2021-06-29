package main

import (
	aw "github.com/deanishe/awgo"
	"jetbrains-project-workflow/pkg/workflow"
)

var wf *aw.Workflow

func init() {
	wf = aw.New()
}

func run() {
	workflow.Open(wf)
}

func main() {
	wf.Run(run)
}
