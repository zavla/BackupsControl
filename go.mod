module BackupsControl

require BackupsControl/dpapi v0.0.1

require BackupsControl/dblist/v2 v2.0.0

require (
	github.com/pkg/profile v1.3.0
	github.com/rsc/benchgraffiti v0.0.0-20170203011114-ff448abfc41f // indirect
	github.com/spf13/cobra v0.0.3
)

replace BackupsControl/dpapi v0.0.1 => ../dpapi

replace BackupsControl/dblist/v2 v2.0.0 => ../../GO/src/dblist/v2
