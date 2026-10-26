package riskmanager

import "errors"

var ErrInvalid = errors.New("riskmanager: invalid")

var ErrInvalidReference = errors.New("riskmanager: invalid reference")

var ErrConflict = errors.New("riskmanager: conflict")

var ErrRiskNotFound = errors.New("riskmanager: risk not found")

var ErrPathwayNotFound = errors.New("riskmanager: pathway not found")

var ErrControlNotFound = errors.New("riskmanager: control not found")

var ErrBarrierNotFound = errors.New("riskmanager: barrier not found")

var ErrActionNotFound = errors.New("riskmanager: action not found")
