// OWNER: stan

package task

import (
	"context"
	"time"

	"github.com/spolu/settle/lib/db"
	"github.com/spolu/settle/lib/errors"
	"github.com/spolu/settle/mint"
	"github.com/spolu/settle/mint/async"
	"github.com/spolu/settle/mint/model"
)

const (
	// TkPropagateOperation propagates an operation
	TkPropagateOperation mint.TkName = "PropagateOperation"
)

func init() {
	async.Registrar[TkPropagateOperation] = NewPropagateOperation
}

// PropagateOperation is in charge of propagating the operation to all required
// mints (up to two other mints, the source's and destination's if they are not
// the same as the operation's owner).
type PropagateOperation struct {
	Client *mint.Client

	ID string
}

// NewPropagateOperation constructs and initializes the task.
func NewPropagateOperation(
	ctx context.Context,
	subject string,
) async.Task {
	return &PropagateOperation{
		ID: subject,
	}
}

// Name returns the task name.
func (t *PropagateOperation) Name() mint.TkName {
	return TkPropagateOperation
}

// Subject returns the task subject.
func (t *PropagateOperation) Subject() string {
	return t.ID
}

// MaxRetries returns the max retries for the task.
func (t *PropagateOperation) MaxRetries() uint {
	return 18
}

// DeadlineForRetry returns the deadline for the provided retry count.
func (t *PropagateOperation) DeadlineForRetry(
	retry uint,
) time.Time {
	return time.Now().Add((1<<retry - 1) * time.Second)
}

// Execute idempotently runs the task to completion or errors.
func (t *PropagateOperation) Execute(
	ctx context.Context,
) error {
	client := &mint.Client{}
	err := client.Init(ctx)
	if err != nil {
		return errors.Trace(err)
	}

	ctx = db.Begin(ctx)
	defer db.LoggedRollback(ctx)

	operation, err := model.LoadCanonicalOperationByID(ctx, t.ID)
	if err != nil {
		return errors.Trace(err)
	} else if operation == nil {
		return errors.Trace(
			errors.Newf("Canonical operation not found: %s", t.ID))
	}

	db.Commit(ctx)

	_, host, err := mint.UsernameAndMintHostFromAddress(ctx,
		operation.Source)
	if err != nil {
		return errors.Trace(err)
	}

	if host != mint.GetHost(ctx) {
		_, err := client.PropagateOffer(ctx, t.ID, host)
		if err != nil {
			return errors.Trace(err)
		}
	}

	_, host, err = mint.UsernameAndMintHostFromAddress(ctx,
		operation.Destination)
	if err != nil {
		return errors.Trace(err)
	}

	if host != mint.GetHost(ctx) {
		_, err := client.PropagateOffer(ctx, t.ID, host)
		if err != nil {
			return errors.Trace(err)
		}
	}

	return nil
}
