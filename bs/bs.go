package main

import (
	"context"

	sbbs "github.com/barbell-math/smoothbrain-bs"
)

func main() {
	sbbs.RegisterBsBuildTarget()
	sbbs.RegisterUpdateDepsTarget()
	sbbs.RegisterGoMarkDocTargets()
	sbbs.RegisterCommonGoCmdTargets(sbbs.GoTargets{
		GenericTestTarget:  true,
		GenericBenchTarget: true,
		GenericFmtTarget:   true,
	})
	sbbs.RegisterMergegateTarget(sbbs.MergegateTargets{
		CheckDepsUpdated:     true,
		CheckReadmeGomarkdoc: true,
		CheckFmt:             true,
		CheckUnitTests:       true,
	})

	sbbs.RegisterTarget(
		context.Background(),
		"race",
		sbbs.Stage(
			"Run go test -race",
			func(ctxt context.Context, cmdLineArgs ...string) error {
				return sbbs.RunStdout(ctxt, "go", "test", "-v", "-race")
			},
		),
	)
	sbbs.Main("bs")
}
