module BackupsControl

require github.com/zavla/dblist/v2 v2.0.0

require (
	github.com/pkg/profile v1.3.0

	//this is special module with its own go.mod file, module name is "github.com/zavla".
	//**** COMMENT THIS LINE BELOW TO USE https://github.com/zavla/... itself.
	github.com/zavla v0.0.0
)

//**** COMMENT LINES BELOW TO USE https://github.com/zavla/... itself.
// Module "github.com/zavla" has its own go.mod file.

// Special replacement for the module dblist/v2.
replace github.com/zavla/dblist/v2 v2.0.0 => ../../GO/src/dblist/v2

// Replacement for special MODULE "github.com/zavla" to use all my _packages_ (not modules) from my local GOPATH
// Directory ../../GO/src has its own go.mod file with "module github.com/zavla" first line.
replace github.com/zavla v0.0.0 => ../../GO/src

//********************************************************************
