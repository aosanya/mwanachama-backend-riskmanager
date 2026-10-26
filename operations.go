package riskmanager

import _ "embed"

//go:embed riskmanager.operations.json
var operationsJSON []byte

func Operations() []byte { return operationsJSON }
