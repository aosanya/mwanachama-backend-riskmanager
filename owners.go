package riskmanager

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/aosanya/mwanachama-backend-shared/identity"
)

type Option func(*riskManager)

func WithOwners(owners identity.Owner, objects ...string) Option {
	return func(m *riskManager) {
		m.owners = owners
		m.ownerObjects = objects
	}
}

func (m *riskManager) ownerChanged(ctx context.Context, what, prior, next string) error {
	if prior == next {
		return nil
	}
	return m.ownerKnown(ctx, what, next)
}

func (m *riskManager) ownerKnown(ctx context.Context, what, owner string) error {
	if m.owners == nil || owner == "" {
		return nil
	}
	object, _, err := m.owners.Resolve(ctx, owner)
	switch {
	case err == nil:
	case errors.Is(err, identity.ErrNotFound), errors.Is(err, identity.ErrNotOwned), errors.Is(err, identity.ErrInvalidRaw):
		return fmt.Errorf("%w: the %s %q is not someone the owner register knows", ErrInvalidReference, what, owner)
	default:
		return err
	}
	if len(m.ownerObjects) > 0 && !slices.Contains(m.ownerObjects, object) {
		return fmt.Errorf("%w: the %s %q is a %s, which cannot own", ErrInvalidReference, what, owner, object)
	}
	return nil
}
