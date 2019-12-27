module BackupsControl

require github.com/zavla/dblist/v2 v2.0.1

require (
	github.com/pkg/profile v1.3.0
	github.com/zavla/dblist v2.0.1+incompatible // indirect
	github.com/zavla/dpapi v1.0.0
	github.com/zavla/sendmail v0.0.0-20190527120221-c154caff6183
	golang.org/x/sys v0.0.0-20191224085550-c709ea063b76 // indirect
)

//**** COMMENT LINES BELOW TO USE https://github.com/zavla/... itself.
// Module "github.com/zavla" has its own go.mod file.

// Special replacement for the module dblist/v2.
//replace github.com/zavla/dblist/v2 v2.0.0 => ../../GO/src/dblist/v2

// Replacement for special MODULE "github.com/zavla" to use all my _packages_ (not modules) from my local GOPATH
// Directory ../../GO/src has its own go.mod file with "module github.com/zavla" first line.
//replace github.com/zavla v0.0.0 => ../../GO/src

//********************************************************************
