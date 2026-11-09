// Copyright (C) MongoDB, Inc. 2026-present.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at http://www.apache.org/licenses/LICENSE-2.0

package csot

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// TimeoutCause is the cancellation cause of a context whose deadline was
// derived by the driver from a named timeout, such as timeoutMS or
// connectTimeoutMS. It wraps context.DeadlineExceeded so that
//
//	errors.Is(context.Cause(ctx), context.DeadlineExceeded)
//
// holds.
type TimeoutCause struct {
	Name    string
	Timeout time.Duration
}

var _ error = &TimeoutCause{}

func (c *TimeoutCause) Error() string {
	return fmt.Sprintf("%s of %v elapsed", c.Name, c.Timeout)
}

// Unwrap returns context.DeadlineExceeded.
func (c *TimeoutCause) Unwrap() error {
	return context.DeadlineExceeded
}

// WithNamedTimeout is like context.WithTimeout, but it records a TimeoutCause
// naming the timeout as the context's cause. If the parent's deadline expires
// first, the parent's cause is retained.
func WithNamedTimeout(
	parent context.Context,
	timeout time.Duration,
	name string,
) (context.Context, context.CancelFunc) {
	return context.WithTimeoutCause(parent, timeout, &TimeoutCause{Name: name, Timeout: timeout})
}

// ContextError returns ctx.Err() annotated with context.Cause(ctx). The result
// always satisfies errors.Is(result, ctx.Err()). If the context has no cause
// beyond ctx.Err(), ctx.Err() is returned unchanged.
func ContextError(ctx context.Context) error {
	err := ctx.Err()
	if err == nil {
		return nil
	}

	cause := context.Cause(ctx)
	if errors.Is(err, cause) {
		return err
	}

	return fmt.Errorf("%w: %w", err, cause)
}

// WrapCause annotates err with context.Cause(ctx) if ctx is done and its cause
// carries more information than ctx.Err(). Use this for errors that a done
// context provoked but that don't come from ctx.Err(), such as net.Dialer or
// net.Conn deadline errors. Otherwise, err is returned unchanged.
func WrapCause(ctx context.Context, err error) error {
	if err == nil || ctx.Err() == nil {
		return err
	}

	cause := context.Cause(ctx)
	if errors.Is(ctx.Err(), cause) || errors.Is(err, cause) {
		return err
	}

	return fmt.Errorf("%w: %w", err, cause)
}
